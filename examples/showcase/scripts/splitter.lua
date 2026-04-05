local data = inputs["in"] or ""
local words = {}
for w in data:gmatch("%S+") do
    words[#words + 1] = w
end
if #words == 0 then
    return { out = "", _display = "(empty)" }
end
local idx = math.floor(_time / 1000) % #words + 1
return { out = words[idx], _display = words[idx] .. " [" .. idx .. "/" .. #words .. "]" }
