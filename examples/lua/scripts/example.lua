-- Example node script: uppercases the input string.

function node:on_event(e)
    local data = e.value or self.inputs["in"]
    if type(data) ~= "string" then
        data = tostring(data or "")
    end
    self:emit("out", string.upper(data))
end
