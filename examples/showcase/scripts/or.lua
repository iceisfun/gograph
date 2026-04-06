-- Type definition
node:set_label("OR")
node:set_category("logic")
node:set_content_height(30)
node:add_input("a", "A", "state")
node:add_input("b", "B", "state")
node:add_output("out", "Output", "state")

local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end

function node:on_change(e)
    local a = truthy(self.inputs.a)
    local b = truthy(self.inputs.b)
    local r = a or b
    self:set("out", r and "1" or "0")
    self:display(r and "1" or "0")
end
