local signal = inputs["in"] or ""
local is_on = signal == "1" or signal == "true" or signal == "on"
if is_on then
    return { on = inputs["in"], _display = "-> ON" }
else
    return { off = inputs["in"], _display = "-> OFF" }
end
