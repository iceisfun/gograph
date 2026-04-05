local period = tonumber(config["period"]) or 2000
local phase = math.floor(time.now() / period) % 2
local on = phase == 0
return { out = on and "1" or "0", _display = on and "ON" or "OFF" }
