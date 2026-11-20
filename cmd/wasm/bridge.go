package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/primissus/ahorita/expr"
	internalsettings "github.com/primissus/ahorita/internal/settings"
)

type runtimeEnv struct {
	now       func() time.Time
	defaultTZ func() string
}

type bridgeOptions struct {
	Now            *string         `json:"now"`
	TZ             *string         `json:"tz"`
	Type           *string         `json:"type"`
	Format         *string         `json:"format"`
	ShowTimezone   json.RawMessage `json:"show_timezone"`
	Units          *string         `json:"units"`
	Explain        *bool           `json:"explain"`
	Lang           *string         `json:"lang"`
	Strict         json.RawMessage `json:"strict"`
	WeekStart      *string         `json:"week_start"`
	WeekOne        *string         `json:"week_one"`
	MonthOverflow  *string         `json:"month_overflow"`
	CountInclusive json.RawMessage `json:"count_inclusive"`
	Country        *string           `json:"country"`
	Weekend        json.RawMessage   `json:"weekend"`
	Calendar       *string           `json:"calendar"`
	DaysOff        []string          `json:"days_off"`
	Aliases        map[string]string `json:"aliases"`
}

func runBridge(input, optsJSON string, rt runtimeEnv) string {
	res, err := evalBridge(input, optsJSON, rt)
	if err != nil {
		res = expr.Result{
			Input: input,
			Error: &expr.Error{
				Pos: -1,
				Msg: err.Error(),
			},
		}
	}
	b, marshalErr := json.Marshal(res)
	if marshalErr != nil {
		fallback, _ := json.Marshal(expr.Result{
			Input: input,
			Error: &expr.Error{
				Pos: -1,
				Msg: marshalErr.Error(),
			},
		})
		return string(fallback)
	}
	return string(b)
}

func evalBridge(input, optsJSON string, rt runtimeEnv) (expr.Result, error) {
	raw, err := parseBridgeOptions(optsJSON)
	if err != nil {
		return expr.Result{}, err
	}

	vals := internalsettings.DefaultValues(nil)

	if err := resolveBridgeTZ(raw.TZ, rt.defaultTZ, &vals); err != nil {
		return expr.Result{}, err
	}

	nowFn := rt.now
	if nowFn == nil {
		nowFn = time.Now
	}
	var now time.Time
	if raw.Now != nil && strings.TrimSpace(*raw.Now) != "" {
		parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(*raw.Now))
		if err != nil {
			return expr.Result{}, fmt.Errorf("invalid now %q: must be RFC 3339 (%w)", *raw.Now, err)
		}
		now = parsed
	} else {
		now = nowFn()
	}

	if err := applyStringSetting(&vals, "type", raw.Type, true); err != nil {
		return expr.Result{}, err
	}
	if err := applyStringSetting(&vals, "format", raw.Format, true); err != nil {
		return expr.Result{}, err
	}
	if err := applyBoolSetting(&vals, "show_timezone", raw.ShowTimezone); err != nil {
		return expr.Result{}, err
	}
	if err := applyStringSetting(&vals, "units", raw.Units, true); err != nil {
		return expr.Result{}, err
	}
	if err := applyStringSetting(&vals, "lang", raw.Lang, true); err != nil {
		return expr.Result{}, err
	}
	if err := applyBoolSetting(&vals, "strict", raw.Strict); err != nil {
		return expr.Result{}, err
	}
	if err := applyStringSetting(&vals, "week_start", raw.WeekStart, true); err != nil {
		return expr.Result{}, err
	}
	if err := applyStringSetting(&vals, "week_one", raw.WeekOne, true); err != nil {
		return expr.Result{}, err
	}
	if err := applyStringSetting(&vals, "month_overflow", raw.MonthOverflow, true); err != nil {
		return expr.Result{}, err
	}
	if err := applyBoolSetting(&vals, "count_inclusive", raw.CountInclusive); err != nil {
		return expr.Result{}, err
	}
	if err := applyStringSetting(&vals, "country", raw.Country, false); err != nil {
		return expr.Result{}, err
	}
	if err := applyWeekendSetting(&vals, raw.Weekend); err != nil {
		return expr.Result{}, err
	}
	if raw.Calendar != nil && strings.TrimSpace(*raw.Calendar) != "" {
		return expr.Result{}, fmt.Errorf("calendar file paths are not supported in browser; pass expressions in days_off")
	}
	if len(raw.Aliases) > 0 {
		vals.Cfg.Aliases = make(map[string]string, len(raw.Aliases))
		for k, v := range raw.Aliases {
			folded, err := expr.ValidateAliasName(k)
			if err != nil {
				return expr.Result{}, err
			}
			vals.Cfg.Aliases[folded] = v
		}
	}

	ctx := expr.Context{
		Now: now,
		Loc: vals.Loc,
		Cfg: vals.Cfg,
	}

	for _, entry := range raw.DaysOff {
		if strings.Contains(entry, "BEGIN:VCALENDAR") {
			days, err := internalsettings.ParseDaysOffICS(entry, ctx)
			if err != nil {
				return expr.Result{}, fmt.Errorf("days_off: %w", err)
			}
			ctx.Cfg.DaysOff = append(ctx.Cfg.DaysOff, days...)
			continue
		}
		days, lineNum, err := internalsettings.ParseDaysOffText(entry, ctx)
		if err != nil {
			return expr.Result{}, fmt.Errorf("days_off line %d: %w", lineNum, err)
		}
		ctx.Cfg.DaysOff = append(ctx.Cfg.DaysOff, days...)
	}

	explain := raw.Explain != nil && *raw.Explain
	return expr.Run(input, ctx, vals.Opts, explain), nil
}

func parseBridgeOptions(optsJSON string) (bridgeOptions, error) {
	trimmed := strings.TrimSpace(optsJSON)
	if trimmed == "" {
		trimmed = "{}"
	}
	if !strings.HasPrefix(trimmed, "{") {
		return bridgeOptions{}, fmt.Errorf("invalid options JSON: expected a JSON object")
	}
	var raw bridgeOptions
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return bridgeOptions{}, fmt.Errorf("invalid options JSON: %w", err)
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return bridgeOptions{}, fmt.Errorf("invalid options JSON: unexpected trailing data")
	}
	return raw, nil
}

func resolveBridgeTZ(explicit *string, defaultTZ func() string, vals *internalsettings.Values) error {
	var tzName string
	if explicit != nil {
		tzName = strings.TrimSpace(*explicit)
	} else {
		if defaultTZ != nil {
			tzName = strings.TrimSpace(defaultTZ())
		}
		if tzName == "" {
			return fmt.Errorf("missing tz: pass an IANA timezone name in options.tz")
		}
	}
	if strings.EqualFold(tzName, "local") {
		return fmt.Errorf("invalid tz %q: must be an IANA timezone name", tzName)
	}
	return internalsettings.Apply(vals, "tz", tzName)
}

func applyStringSetting(vals *internalsettings.Values, key string, val *string, skipEmpty bool) error {
	if val == nil {
		return nil
	}
	if skipEmpty && *val == "" {
		return nil
	}
	return internalsettings.Apply(vals, key, *val)
}

func applyBoolSetting(vals *internalsettings.Values, key string, raw json.RawMessage) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var b bool
	if err := json.Unmarshal(raw, &b); err == nil {
		return internalsettings.Apply(vals, key, strconv.FormatBool(b))
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return internalsettings.Apply(vals, key, s)
	}
	return fmt.Errorf("invalid %s: expected boolean", key)
}

func applyWeekendSetting(vals *internalsettings.Values, raw json.RawMessage) error {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return internalsettings.Apply(vals, "weekend", s)
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		if len(list) == 0 {
			return internalsettings.Apply(vals, "weekend", "none")
		}
		return internalsettings.Apply(vals, "weekend", strings.Join(list, ","))
	}
	return fmt.Errorf("invalid weekend: expected weekday list or string")
}
