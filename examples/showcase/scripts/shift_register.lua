local bits = tonumber(config["bits"]) or 8
local speed = tonumber(config["speed"]) or 500
local clock = inputs["in"]
local active = clock == "1" or clock == "on" or clock == "true"

if active then
    local shift = math.floor(time.now() / speed) % bits
    local display = string.rep("0", shift) .. "1" .. string.rep("0", bits - shift - 1)
    return { out = "1", _display = display }
else
    return { out = "0", _display = string.rep("-", tonumber(config["bits"]) or 8) }
end
