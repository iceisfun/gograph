-- Type definition
node:set_label("Rate Limit")
node:set_category("transform")
node:set_content_height(30)
node:add_input("in", "Input", "any")
node:add_output("out", "Output", "any")
node:define_config("rate", "2000", "Rate (ms)")

function node:on_init()
    self.state.qh = 1
    self.state.qt = 1
    self.state.last_emit = 0
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
        -- On cooldown — schedule a tick for when it expires.
        self:schedule_tick(rate - elapsed)
        -- self:display("HOLD [" .. size .. "]")
        self:display("HOLD [" .. size .. "] @" .. self.config.rate .. "ms")
    else
        self:display("IDLE")
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
        self:display("HOLD [" .. size .. "]")
    end
end

function node:drain()
    local val = self.state[self.state.qh]
    self.state[self.state.qh] = nil
    self.state.qh = self.state.qh + 1
    self.state.last_emit = time.now()

    self:emit("out", val)

    local size = self.state.qt - self.state.qh
    self:display("EMIT [" .. size .. "]")

    -- More queued? Schedule next drain after cooldown.
    if size > 0 then
        local rate = tonumber(self.config.rate) or 2000
        self:schedule_tick(rate)
    end
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
