-- Type definition
node:set_label("Oscillator")
node:set_category("source")
node:set_content_height(30)
node:add_output("out", "Output", "state")
node:define_config("period", "2000", "Period (ms)")

function node:update_title()
    local ms = tonumber(self.config.period) or 2000
    self:set_label("Osc " .. ms .. "ms")
end

function node:on_init()
    self:init_tick(tonumber(self.config.period) or 2000)
    self:update_title()
end

function node:on_config()
    self:init_tick(tonumber(self.config.period) or 2000)
    self:update_title()
end

function node:on_tick()
    local period = tonumber(self.config.period) or 2000
    local phase = math.floor(time.now() / period) % 2
    local on = phase == 0
    self:set("out", on and "1" or "0")
    self:display(on and "ON" or "OFF")
end
