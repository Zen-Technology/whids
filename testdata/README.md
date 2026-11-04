# Test fixtures

`events.json` is newline-delimited `event.EdrEvent` JSON used by the `event`,
`logger` and `api/server` test suites. It is derived from `agent/test/events.json`
(real Sysmon/ETW captures) with the legacy `GeneInfo` field replaced by the
current `Detection` field. Every 4th event carries a detection so the alert,
report and forwarding tests have alerts to work with.

Upstream relied on an uncommitted `./data/events.json` per package, which made
the suites panic on a clean checkout.
