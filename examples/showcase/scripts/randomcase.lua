local data = inputs["in"]
if type(data) ~= "string" then data = tostring(data or "") end
local result = ""
for i = 1, #data do
    local c = string.sub(data, i, i)
    if math.random() > 0.5 then
        result = result .. string.upper(c)
    else
        result = result .. string.lower(c)
    end
end
return { out = result, _display = result }
