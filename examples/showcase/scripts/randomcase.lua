-- Type definition
node:set_label("RanDOmCaSe")
node:set_category("transform")
node:set_content_height(40)
node:add_input("in", "Input", "string")
node:add_output("out", "Output", "string")

function node:on_event(e)
    local data = e.value or self.inputs["in"]
    if type(data) ~= "string" then data = tostring(data or "") end
    local result = ""
    for i = 1, #data do
        local c = string.sub(data, i, i)
        if math.random() > 0.5 then
            result = result .. string.upper(c)
        else
            result = result .. string.lower(c)
        end
    end
    self:emit("out", result)
    self:display(result)
end
