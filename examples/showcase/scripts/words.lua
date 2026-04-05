local phrases = {
    "Hello World", "GoGraph is alive", "Lua powers this",
    "Sensor online", "Data flowing", "Box detected",
    "Gate open", "Belt running", "Divert left",
    "Merge complete", "Signal high", "Batch ready",
    "Inspection pass", "Label applied", "Weight OK",
    "Scan complete", "Route alpha", "Queue empty",
    "Heartbeat", "System nominal",
}
local idx = math.floor(_time / 5000) % #phrases + 1
return { out = phrases[idx], _display = phrases[idx] }
