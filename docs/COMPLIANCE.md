TASK: Audit Go Conky implementation for functional correctness and compatibility. Generate AUDIT.md report.

EXECUTION MODE: Report Generation - test features, document bugs, output to AUDIT.md

AUDIT PROCESS:

For each feature: DISCOVER location → TEST functionality → VERIFY against original Conky → DOCUMENT bugs

SCOPE:

1. **Configuration (150+ directives)**: alignment, own_window_*, fonts, colors, spacing, update_interval, etc.
2. **Cairo Functions (50+)**: drawing, paths, text, patterns, transforms, compositing
3. **Lua Bridge**: conky_parse(), conky_window table, hooks (main, shutdown, draw)
4. **Display Objects (200+)**: ${cpu}, ${mem}, ${time}, ${exec}, ${top}, ${battery}, graphs, bars
5. **Window Management**: ARGB transparency, positioning, hints, types
6. **System Monitoring**: /proc parsing, sensor data, network stats, disk I/O

TEST METHODOLOGY:

**Config Directives:**
- Valid value: Does it parse and apply correctly?
- Invalid value: Graceful error handling?
- Edge cases: Empty, extreme values, special chars?
- Interactions: Works with related settings?

**Cairo Rendering:**
- Execute Lua test drawing known shapes/colors
- Verify pixel-perfect output (coordinates, colors, dimensions)
- Check memory cleanup (no leaks)

**Display Objects:**
- Compare output to ground truth (ps, free, /proc/stat, date commands)
- Test update intervals and caching
- Validate formatting and units
- Error handling for missing data sources

**Example Tests:**

```
TEST: ${cpu} accuracy
Action: Generate 100% CPU load
Expected: Value → 100% (±5%)
Verify: Compare to /proc/stat calculation
Status: [PASS/FAIL + actual value]

TEST: own_window_argb_visual
Setup: argb_visual=true, argb_value=128
Verify: 32-bit visual, 50% transparency renders correctly
Check: xprop _NET_WM_WINDOW_TYPE, visual depth
Status: [PASS/FAIL + screenshot/measurement]

TEST: conky_parse()
Input: "CPU: ${cpu}%"
Expected: "CPU: 45%" (actual percentage)
Status: [PASS/FAIL + output]
```

BUG REPORT FORMAT:
```
BUG-NNN: [Title]
Severity: Critical/High/Medium/Low
Feature: [Name]
Reproduce: [Steps]
Expected: [Correct behavior]
Actual: [What happens]
Location: [file:line if found]
Fix: [Suggested approach]
```

OUTPUT TO AUDIT.md:
```markdown
# Conky Go Implementation Audit

## Summary
- Date: [date]
- Tests: XXX total, XXX passed, XXX failed
- Bugs: XX critical, XX high, XX medium, XX low
- Compatibility: XX%

## Test Results by Category

### Configuration (XX% pass rate)
| Directive | Status | Bugs | Notes |
|-----------|--------|------|-------|
| alignment | ✅ PASS | 0 | |
| own_window_argb_visual | ❌ FAIL | BUG-001 | Transparency broken |
| update_interval | ✅ PASS | 0 | |

### Cairo Rendering (XX% pass rate)
| Function | Status | Bugs | Notes |
|----------|--------|------|-------|
| cairo_arc | ⚠️ PARTIAL | BUG-002 | Coordinates off by 1px |
| cairo_show_text | ✅ PASS | 0 | |

### Lua Integration (XX% pass rate)
| Feature | Status | Bugs | Notes |
|---------|--------|------|-------|
| conky_parse | ❌ FAIL | BUG-003 | Crashes on ${exec} |
| conky_window | ✅ PASS | 0 | |

### Display Objects (XX% pass rate)
| Object | Accuracy | Bugs | Notes |
|--------|----------|------|-------|
| ${cpu} | ±2% | 0 | ✅ Verified vs /proc/stat |
| ${mem} | ±5% | BUG-004 | Wrong calculation |
| ${time} | Exact | 0 | ✅ |
| ${exec} | N/A | BUG-005 | Not implemented |

### Window Management (XX% pass rate)
### System Monitoring (XX% pass rate)

## Bugs Found

### Critical (Blockers)
**BUG-001: ARGB transparency not working**
- Severity: Critical
- Feature: own_window_argb_visual
- Reproduce: Set argb_visual=true, argb_value=128
- Expected: 50% transparent window
- Actual: Opaque window, wrong visual depth (24 not 32)
- Location: window.go:145
- Fix: Use XMatchVisualInfo(depth=32, class=TrueColor)

### High Priority
### Medium Priority
### Low Priority

## Compatibility Matrix

| Category | Features | Working | Broken | Missing | Score |
|----------|----------|---------|--------|---------|-------|
| Config | 150 | 140 | 5 | 5 | 93% |
| Cairo | 50 | 45 | 3 | 2 | 90% |
| Lua | 10 | 6 | 2 | 2 | 60% |
| Objects | 200 | 180 | 15 | 5 | 90% |
| Window | 20 | 15 | 3 | 2 | 75% |
| Monitoring | 30 | 28 | 1 | 1 | 93% |
| **Overall** | **460** | **414** | **29** | **17** | **90%** |

## Fix Priority

**Must Fix (Blockers):**
1. BUG-001: ARGB transparency (2h)
2. BUG-003: conky_parse crashes (4h)

**Should Fix:**
1. BUG-002: Cairo coordinate offset (1h)
2. BUG-004: Memory calculation (2h)

**Can Defer:**
1. BUG-005: Missing ${exec} (8h)

## Recommendations

1. Fix critical transparency bug before any release
2. Implement missing Lua integration (60% complete)
3. Add automated regression test suite
4. Performance: Current CPU usage 2% (target <1%)
```

VALIDATION CHECKLIST:
✓ Test every config directive with valid/invalid inputs
✓ Verify Cairo output visually (pixel checks)
✓ Compare display objects to system ground truth
✓ Check error handling (malformed config, missing files)
✓ Measure performance (CPU, memory, render time)
✓ Document all bugs with reproduction steps
✓ Calculate compatibility percentage
✓ Prioritize fixes with effort estimates

CONSTRAINTS:
- Test systematically, don't skip features
- Provide actual test results, not just plans
- Include evidence (values, measurements, screenshots)
- Keep bug reports concise but actionable
- Focus on compatibility with original Conky behavior

OUTPUT: Create AUDIT.md with complete test results, bug inventory, and fix roadmap.
