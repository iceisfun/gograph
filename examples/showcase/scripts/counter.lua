local period = tonumber(config["period"]) or 5000
local count = math.floor(time.now() / period) % 256
return { out = tostring(count), _display = tostring(count) }
