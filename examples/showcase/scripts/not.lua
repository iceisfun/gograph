local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end
local a = truthy(inputs["in"])
local r = not a
return { out = r and "1" or "0", _display = r and "1" or "0" }
