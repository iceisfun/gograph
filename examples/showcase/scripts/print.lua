-- Type definition
node:set_label("Print")
node:set_category("output")
node:add_input("in", "Input", "string")

function node:on_event(e)
    local data = e.value or self.inputs["in"]
    if type(data) ~= "string" then data = tostring(data or "") end
    print("[print] " .. data)
end
