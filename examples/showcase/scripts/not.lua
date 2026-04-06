-- Type definition
node:set_label("NOT")
node:set_category("logic")
node:set_content_height(30)
node:add_input("in", "Input", "string")
node:add_output("out", "Output", "string")

local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end

function node:on_event(e)
    local a = truthy(e.value or self.inputs["in"])
    local r = not a
    self:emit("out", r and "1" or "0")
    self:display(r and "1" or "0")
end
