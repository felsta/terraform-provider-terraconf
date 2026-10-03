data "terraconf_yaml" "config" {
  input_glob = "./data/virtual_machines.yml"
}

resource "hcloud_server" "virtual_machine" {
  for_each = {
    for vm in data.terraconf_yaml.config.output
    : vm.name => vm
  }

  name        = each.key
  image       = each.value.image
  server_type = each.value.sku

  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }

  // By default, the output contains '_terraconf' and
  // '_terraconf_path' keys in all objects to facilitate error
  // reporting. sanatize() removes these annotations (in this
  // case to avoid setting labels named '_terraconf' and
  // '_terraconf_path' on our VM)
  labels = provider::terraconf::sanatize(each.value.labels)
}
