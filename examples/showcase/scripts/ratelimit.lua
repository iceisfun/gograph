-- Type definition
node:set_label("Rate Limit")
node:set_category("transform")
node:set_content_height(50)
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")
node:define_config("rate", "2000", "Rate (ms)")

function node:update_title()
    local ms = tonumber(self.config.rate) or 2000
    self:set_label("Rate Limit " .. ms .. "ms")
end

function node:on_init()
    self.state.qh = 1
    self.state.qt = 1
    self.state.last_emit = 0
    self:update_title()
end

function node:on_event(e)
    local rate = tonumber(self.config.rate) or 2000

    -- Enqueue on arrival.
    if e.type == "arrival" and e.value then
        self.state[self.state.qt] = e.value
        self.state.qt = self.state.qt + 1
    end

    -- Try to drain immediately.
    local now = time.now()
    local elapsed = now - self.state.last_emit
    local size = self.state.qt - self.state.qh

    if size > 0 and elapsed >= rate then
        self:drain()
    elseif size > 0 then
        self:schedule_tick(rate - elapsed)
        self:display("status", "HOLD [" .. size .. "] @" .. self.config.rate .. "ms")
    else
        self:display("status", "IDLE")
        self:display("value", "")
    end
end

function node:on_tick()
    local rate = tonumber(self.config.rate) or 2000
    local now = time.now()
    local elapsed = now - self.state.last_emit
    local size = self.state.qt - self.state.qh

    if size > 0 and elapsed >= rate then
        self:drain()
    elseif size > 0 then
        self:schedule_tick(rate - elapsed)
        self:display("status", "HOLD [" .. size .. "] @" .. self.config.rate .. "ms")
    end
end

function node:drain()
    local val = self.state[self.state.qh]
    self.state[self.state.qh] = nil
    self.state.qh = self.state.qh + 1
    self.state.last_emit = time.now()

    self:emit("out", val)

    local size = self.state.qt - self.state.qh
    self:display("status", "EMIT [" .. size .. "]")
    self:display("value", tostring(val), {color="#00ff00", animate="flash", duration=300})

    -- More queued? Schedule next drain after cooldown.
    if size > 0 then
        local rate = tonumber(self.config.rate) or 2000
        self:schedule_tick(rate)
    end
end

function node:on_config()
    self:update_title()
end

function node:on_disconnect(e)
    self.state.qh = 1
    self.state.qt = 1
end

function node:on_shutdown()
    local size = self.state.qt - self.state.qh
    if size > 0 then
        self:log("shutting down with " .. size .. " queued items")
    end
end
