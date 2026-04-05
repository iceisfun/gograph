local period = tonumber(config["period"]) or 5000
local count = math.floor(_time / period) % 256
return { out = tostring(count), _display = tostring(count) }
