-- Type definition
node:set_label("Gate")
node:set_category("transform")
node:set_interactive(true)
node:set_content_height(40)
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")

function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
end

function node:on_event(e)
    if self.config.state == "on" then
        self:emit("out", self.inputs["in"])
        self:display("OPEN")
    else
        self:display("CLOSED")
    end
end
