-- Type definition
node:set_label("Delay")
node:set_category("delay")
node:set_content_height(30)
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")
node:define_config("duration", "0", "Duration (ms)")

function node:queue_size()
    return self.state.qt - self.state.qh
end

function node:update_status()
    local ms = tonumber(self.config.duration) or 0
    local size = self:queue_size()
    self:set_label("Delay " .. ms .. "ms")
    if size > 0 then
        self:display("queue", "queued: " .. size, {color="#ffaa00", animate="flash", duration=300})
        self:glow(math.min(size * 500, 5000))
    else
        self:display("queue", "")
    end
end

function node:on_init()
    self.state.qh = 1
    self.state.qt = 1
    self:update_status()
end

function node:on_config()
    self:update_status()
end

function node:on_event(e)
    local ms = tonumber(self.config.duration) or 0
    local val = e.value or self.inputs["in"]
    self.state[self.state.qt] = { value = val, at = time.now() + ms }
    self.state.qt = self.state.qt + 1
    self:update_status()
    self:schedule_tick(ms)
end

function node:on_tick()
    local now = time.now()
    while self.state.qh < self.state.qt do
        local entry = self.state[self.state.qh]
        if entry.at > now then
            self:schedule_tick(entry.at - now)
            self:update_status()
            return
        end
        self:emit("out", entry.value)
        self.state[self.state.qh] = nil
        self.state.qh = self.state.qh + 1
    end
    self:update_status()
end
