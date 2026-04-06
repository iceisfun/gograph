-- Type definition
node:set_label("Shift Register")
node:set_category("logic")
node:set_content_height(30)
node:add_input("clk", "Clock", "any")
node:add_output("b0", "Bit 0", "string")
node:add_output("b1", "Bit 1", "string")
node:add_output("b2", "Bit 2", "string")
node:add_output("b3", "Bit 3", "string")
node:add_output("b4", "Bit 4", "string")
node:add_output("b5", "Bit 5", "string")
node:add_output("b6", "Bit 6", "string")
node:add_output("b7", "Bit 7", "string")
node:define_config("bits", "8", "bits")
node:define_config("speed", "500", "Speed (ms)")

function node:on_event(e)
    local bits = tonumber(self.config.bits) or 8
    local speed = tonumber(self.config.speed) or 500
    local clock = self.inputs.clk
    local active = clock == "1" or clock == "on" or clock == "true"

    local now = time.now()
    local pos
    if active then
        pos = math.floor(now / speed) % bits
    else
        pos = math.floor(now / (speed * bits)) * 1 % bits
    end

    local display = {}
    for i = 0, bits - 1 do
        local val = (i == pos) and "1" or "0"
        self:emit("b" .. i, val)
        display[#display + 1] = val
    end
    self:display(table.concat(display))
end
