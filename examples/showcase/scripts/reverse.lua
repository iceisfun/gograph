local data = inputs["in"]
if type(data) ~= "string" then data = tostring(data or "") end
return { out = string.reverse(data) }
