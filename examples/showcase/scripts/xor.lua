local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end
local a = truthy(inputs["a"])
local b = truthy(inputs["b"])
local r = (a and not b) or (not a and b)
return { out = r and "1" or "0", _display = r and "1" or "0" }
