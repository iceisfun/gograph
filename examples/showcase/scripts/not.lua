-- Type definition
node:set_label("NOT")
node:set_category("logic")
node:set_content_height(30)
node:add_input("in", "Input", "state")
node:add_output("out", "Output", "state")

local function truthy(v)
    return v == "1" or v == "true" or v == "on"
end

function node:on_change(e)
    local a = truthy(self.inputs["in"])
    local r = not a
    self:set("out", r and "1" or "0")
    self:display(r and "1" or "0")
end
