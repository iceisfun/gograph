local rate = tonumber(config["rate"]) or 2000
local window = math.floor(time.now() / rate)
if window % 2 == 0 then
    return { out = inputs["in"], _display = "PASS" }
else
    return { _display = "HOLD" }
end
