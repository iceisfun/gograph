-- Type definition
node:set_label("Delay")
node:set_category("delay")
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")
node:define_config("duration", "0", "Duration (ms)")

function node:on_event(e)
    self:emit("out", e.value or self.inputs["in"])
end
