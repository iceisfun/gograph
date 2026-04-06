-- Type definition
node:set_label("Switch")
node:set_category("transform")
node:set_content_height(30)
node:add_input("en", "Enable", "state")
node:add_input("in", "Data", "any")
node:add_output("out", "Output", "any")
node:add_output("discard", "Discard", "any")

function node:update_display()
    local enabled = self.inputs.en == "1" or self.inputs.en == "true" or self.inputs.en == "on"
    self:display(enabled and "OPEN" or "CLOSED")
end

function node:on_change(e)
    self:update_display()
end

function node:on_event(e)
    local enabled = self.inputs.en == "1" or self.inputs.en == "true" or self.inputs.en == "on"
    local val = e.value or self.inputs["in"]
    if enabled then
        self:emit("out", val)
    else
        self:emit("discard", val)
    end
    self:update_display()
end
