-- Type definition
node:set_label("Show")
node:set_category("output")
node:set_content_height(40)
node:add_input("in", "Input", "any")

function node:on_event(e)
    local data = e.value or self.inputs["in"]
    if type(data) ~= "string" then data = tostring(data or "") end
    self:display(data)
end

function node:on_change(e)
    local data = e.value or self.inputs["in"]
    if type(data) ~= "string" then data = tostring(data or "") end
    self:display(data)
end
