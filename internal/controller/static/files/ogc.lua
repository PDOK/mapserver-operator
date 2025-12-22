if lighty.r.req_attr["request.method"] == "GET" then

    -- obtain service type from environment
    serviceType = os.getenv('SERVICE_TYPE'):lower()

    path = lighty.r.req_attr["uri.path"]
    query = lighty.r.req_attr["uri.query"]

    -- handle legend requests
    if serviceType == "wms" then
        _, _, file = path:find(".*/legend/(.*)")
        if file then
            if file:find(".*%.png") then
                local legendPath = "/var/www/legend/" .. file
                local stat = lighty.stat(legendPath)
                if (not stat or not stat.is_file) then
                    -- don't serve non existing legend file
                    return 404
                end
                lighty.content = { { filename = legendPath } }
                lighty.header['Content-Type'] = "image/png"
                return 200
            end

            return 404
        end
    end

    params = {}
    if query then
        for k, v in query:gmatch("([^?&=]+)=([^&]+)") do
            k = k:lower()

            params[k] = v
        end
    end

    -- assign service and version default values
    version = params['version']
    service = params['service']

    if not service then
        service = serviceType
    else
        service = service:lower()
    end

    if (service == 'wms' and (not version or version ~= '1.1.1')) then
        version = '1.3.0'
    end

    if (service == 'wfs' and (not version or (version ~= '1.0.0' and version ~= '1.1.0'))) then
        version = '2.0.0'
    end

    -- serve static content
    request = params['request']
    if request then
        request = request:lower()

        staticStatus = 200
        staticContentType = 'text/xml; charset=UTF-8'
        if request == 'getcapabilities' then
            if (service == 'wms' and version == '1.3.0') then
                staticFile = '/var/www/config/capabilities_wms_130.xml'
            elseif (service == 'wfs' and version == '2.0.0') then
                staticFile = '/var/www/config/capabilities_wfs_200.xml'
            end
        elseif service == 'wfs' and request == 'getfeature' then
            startindex = params['startindex']
            if startindex and tonumber(startindex) > 50000 then
                staticFile = '/srv/mapserver/config/scraping-error.xml'
                staticStatus = 400
            end
        end

        if staticFile then
            lighty.content = { { filename = staticFile } }
            lighty.header['Content-Type'] = staticContentType
            return staticStatus
        end
    end
end