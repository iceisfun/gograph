-- Type definition
node:set_label("Toggle")
node:set_category("source")
node:set_interactive(true)
node:set_content_height(40)
node:add_output("out", "Output", "state")

function node:update_state()
    local on = self.config.state == "on"
    self:set("out", on and "1" or "0")
    self:display(on and "ON" or "OFF")
end

function node:on_init()
    self:update_state()
end

function node:on_click()
    if self.config.state == "on" then
        self:set_config("state", "off")
    else
        self:set_config("state", "on")
    end
    self:update_state()
end

function node:on_config()
    self:update_state()
end
