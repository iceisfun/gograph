-- Type definition
node:set_label("Counter")
node:set_category("source")
node:set_content_height(30)
node:add_output("out", "Output", "string")
node:define_config("period", "5000", "Period (ms)")

function node:on_init()
    self:init_tick(tonumber(self.config.period) or 5000)
    self.state.count = 0
end

function node:on_config()
    self:init_tick(tonumber(self.config.period) or 5000)
end

function node:on_tick()
    self.state.count = (self.state.count or 0) + 1
    local count = self.state.count % 256
    self:emit("out", tostring(count))
    self:display(tostring(count))
end
