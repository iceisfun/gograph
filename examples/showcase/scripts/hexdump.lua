local data = inputs["in"]
if type(data) ~= "string" then data = tostring(data or "") end
local hex = {}
for i = 1, #data do
    hex[#hex + 1] = string.format("%02x", string.byte(data, i))
end
local result = table.concat(hex, " ")
print("[hexdump] " .. result)
return {}
