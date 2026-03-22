DOCKER_VAR_DIR = '/var/lib/docker/containers/'
DOCKER_CONTAINER_CONFIG_FILE = '/config.v2.json'
CACHE_TTL_SEC = 300
DOCKER_CONTAINER_METADATA = {
  ['docker.container_name'] = '\"Name\":\"/?(.-)\"',
  ['docker.container_image'] = '\"Image\":\"/?(.-)\"',
  ['docker.container_started'] = '\"StartedAt\":\"/?(.-)\"'
}
cache = {}

-- record tag should countain docker log file path
-- last part of log file name is container id
function get_container_id_from_tag(tag)
  return tag:match'^.+/(.-)-json_log$'
end

-- Gets metadata from config.v2.json file for container
function get_container_metadata_from_disk(container_id)
  local docker_config_file = DOCKER_VAR_DIR .. container_id .. DOCKER_CONTAINER_CONFIG_FILE
  fl = io.open(docker_config_file, 'r')
  if fl == nil then
    return nil
  end

  -- parse json file and create record for cache
  local data = { time = os.time() }
  for line in fl:lines() do
    for key, regex in pairs(DOCKER_CONTAINER_METADATA) do
      local match = line:match(regex)
      if match then
        data[key] = match
      end
    end
  end
  fl:close()

  if next(data) == nil then
    return nil
  else
    return data
  end
end

function encrich_with_docker_metadata(tag, timestamp, record)
  -- Get container id from tag
  container_id = get_container_id_from_tag(tag)
  if not container_id then
    return 0, 0, 0
  end

  -- Add container_id to record
  new_record = record
  new_record['docker.container_id'] = container_id

  -- Check if we have fresh cache record for container
  local cached_data = cache[container_id]
  if cached_data == nil or ( os.time() - cached_data['time'] > CACHE_TTL_SEC) then
    cached_data = get_container_metadata_from_disk(container_id)
    cache[container_id] = cached_data
    new_record['source'] = 'disk' -- for troubleshooting only
  else
    new_record['source'] = 'cache' -- for troubleshooting only
  end

  -- Metadata found in cache or got from disk, enrich record
  if cached_data then
    for key, regex in pairs(DOCKER_CONTAINER_METADATA) do
      new_record[key] = cached_data[key]
    end
  end

  return 1, timestamp, new_record
end