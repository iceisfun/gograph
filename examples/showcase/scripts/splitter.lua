-- Type definition
node:set_label("Splitter")
node:set_category("transform")
node:set_content_height(30)
node:add_input("in", "Input", "string")
node:add_output("out", "Output", "string")

function node:on_event(e)
    local data = e.value or self.inputs["in"] or ""

    local words = {}
    for w in data:gmatch("%S+") do
        words[#words + 1] = w
    end

    if #words == 0 then
        return
    end

    -- Emit each word as a separate event down the connection.
    for _, w in ipairs(words) do
        self:emit("out", w)
    end
    self:display(#words .. " words")
end
