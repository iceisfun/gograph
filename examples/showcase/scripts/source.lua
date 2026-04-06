-- Type definition
node:set_label("Source")
node:set_category("source")
node:set_content_height(30)
node:add_output("out", "Output", "string")
node:define_config("message", "Hello, World!", "message")
node:define_config("interval", "5000", "interval")

function node:on_init()
    self:init_tick(tonumber(self.config.interval) or 5000)
end

function node:on_config()
    self:init_tick(tonumber(self.config.interval) or 5000)
end

function node:on_tick()
    local msg = self.config.message or "Hello, World!"
    self:emit("out", msg)
    self:display(msg)
end
