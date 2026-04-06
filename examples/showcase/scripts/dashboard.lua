-- Type definition
node:set_label("Dashboard")
node:set_category("output")
node:set_content_height(200)
node:add_input("in", "Input", "any")
node:define_config("interval", "2000", "Update interval (ms)")

function node:on_init()
    self.state.step = 0
    self.state.history = {}
    self:init_tick(tonumber(self.config.interval) or 2000)
    self:display("status", { type="badge", text="INIT", color="#fff", background="#3498db" })
    self:display("leds", { type="led", states={false, false, false, false, false, false, false, false} })
    self:display("progress", { type="progress", value=0 })
    self:display("load", { type="spinner", visible=false })
end

function node:on_config()
    self:init_tick(tonumber(self.config.interval) or 2000)
end

function node:on_tick()
    self.state.step = (self.state.step or 0) + 1
    local step = self.state.step

    -- Progress: cycle 0..1 over 8 steps
    local progress = (step % 8) / 8
    local interval = tonumber(self.config.interval) or 2000
    self:display("progress", { type="progress", value=progress, duration=interval, color="#4CAF50" })

    -- LEDs: binary counter
    local leds = {}
    for i = 0, 7 do
        leds[i + 1] = (math.floor(step / (2 ^ i)) % 2) == 1
    end
    self:display("leds", { type="led", states=leds, color="#00ffcc" })

    -- Badge: alternating status
    if step % 4 == 0 then
        self:display("status", { type="badge", text="OK", color="#fff", background="#2ecc71" })
    elseif step % 4 == 1 then
        self:display("status", { type="badge", text="BUSY", color="#fff", background="#f39c12" })
    elseif step % 4 == 2 then
        self:display("status", { type="badge", text="WARN", color="#fff", background="#e67e22" })
    else
        self:display("status", { type="badge", text="ERR", color="#fff", background="#e74c3c" })
    end

    -- Sparkline: sine wave history
    local val = math.sin(step * 0.3) * 50 + 50
    local history = self.state.history or {}
    history[#history + 1] = val
    if #history > 20 then
        local trimmed = {}
        for i = #history - 19, #history do
            trimmed[#trimmed + 1] = history[i]
        end
        history = trimmed
    end
    self.state.history = history
    self:display("spark", { type="sparkline", values=history, min=0, max=100 })

    -- Spinner: active on odd steps
    self:display("load", { type="spinner", visible=(step % 2 == 1), color="#3498db" })
end

function node:on_event(e)
    self:glow(300)
end
