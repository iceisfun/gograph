-- Type definition
node:set_label("Oscillator")
node:set_category("source")
node:set_content_height(30)
node:add_output("out", "Output", "string")
node:define_config("period", "2000", "Period (ms)")

function node:on_init()
    self:init_tick(tonumber(self.config.period) or 2000)
end

function node:on_config()
    self:init_tick(tonumber(self.config.period) or 2000)
end

function node:on_tick()
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    local on = phase == 0
    self:emit("out", on and "1" or "0")
    self:display(on and "ON" or "OFF")
end
