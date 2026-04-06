-- Type definition
node:set_label("Switch")
node:set_category("transform")
node:set_content_height(30)
node:add_input("in", "Input", "string")
node:add_output("on", "On", "string")
node:add_output("off", "Off", "string")

function node:on_event(e)
    local signal = e.value or self.inputs["in"] or ""
    local is_on = signal == "1" or signal == "true" or signal == "on"
    if is_on then
        self:emit("on", signal)
        self:display("-> ON")
    else
        self:emit("off", signal)
        self:display("-> OFF")
    end
end
