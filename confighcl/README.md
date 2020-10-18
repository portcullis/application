# Config for HCL

This is based on [gohcl](https://github.com/hashicorp/hcl/tree/hcl2/gohcl). The primary difference is that this will instead handle partials by default to allow for more dynamic based configurations without all the top level schema defintions being required.

## Additionally, this supports both `hcl` and `config` tags with the same values.
