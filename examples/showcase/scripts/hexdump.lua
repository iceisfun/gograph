-- Type definition
node:set_label("Hex Dump")
node:set_category("output")
node:add_input("in", "Input", "string")

function node:on_event(e)
    local data = e.value or self.inputs["in"]
    if type(data) ~= "string" then data = tostring(data or "") end
    local hex = {}
    for i = 1, #data do
        hex[#hex + 1] = string.format("%02x", string.byte(data, i))
    end
    print("[hexdump] " .. table.concat(hex, " "))
end
