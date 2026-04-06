-- Type definition
node:set_label("Words")
node:set_category("source")
node:set_content_height(30)
node:add_output("out", "Output", "string")
node:define_config("delay", "5000", "Delay (ms)")

local phrases = {
    "Hello World", "GoGraph is alive", "Lua powers this",
    "Sensor online", "Data flowing", "Box detected",
    "Gate open", "Belt running", "Divert left",
    "Merge complete", "Signal high", "Batch ready",
    "Inspection pass", "Label applied", "Weight OK",
    "Scan complete", "Route alpha", "Queue empty",
    "Heartbeat", "System nominal",
    "The quick brown fox jumps over the lazy dog",
    "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
    "To be or not to be, that is the question",
    "All your base are belong to us",
    "Fortune rides like the sun on high with the fox that makes the ravens fly.",
    "There can be no health in us, nor any good thing grow.",
    "Soul of Fire, heart of stone, in pride he conquers forcing the proud to yield.",
}

function node:on_init()
    self.state.idx = 0
    self:init_tick(tonumber(self.config.delay) or 5000)
end

function node:on_config()
    self:init_tick(tonumber(self.config.delay) or 5000)
end

function node:on_tick()
    self.state.idx = ((self.state.idx or 0) % #phrases) + 1
    self:emit("out", phrases[self.state.idx])
    self:display(phrases[self.state.idx])
end
