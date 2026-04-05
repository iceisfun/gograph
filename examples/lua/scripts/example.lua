-- Example node script: uppercases the input string.
-- Receives: inputs["in"] (string)
-- Returns: { out = <uppercased string> }

local data = inputs["in"]
if type(data) ~= "string" then
    data = tostring(data or "")
end

return { out = string.upper(data) }
