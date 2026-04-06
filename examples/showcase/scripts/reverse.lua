-- Type definition
node:set_label("Reverse")
node:set_category("transform")
node:add_input("in", "Input", "string")
node:add_output("out", "Output", "string")

function node:on_event(e)
    local data = e.value or self.inputs["in"]
    if type(data) ~= "string" then data = tostring(data or "") end
    self:emit("out", string.reverse(data))
end
