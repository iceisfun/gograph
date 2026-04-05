local bits = tonumber(config["bits"]) or 8
local shift = math.floor(time.now() / 500) % bits
local display = string.rep("0", shift) .. "1" .. string.rep("0", bits - shift - 1)
return { out = inputs["in"] or display, _display = display }
