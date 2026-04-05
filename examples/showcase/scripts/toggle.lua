local state = config["state"] or "off"
local on = state == "on"
return { out = on and "1" or "0", _display = on and "ON" or "OFF" }
