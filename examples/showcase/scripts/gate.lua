local state = config["state"] or "on"
if state == "on" then
    return { out = inputs["in"], _display = "OPEN" }
else
    return { _display = "CLOSED" }
end
