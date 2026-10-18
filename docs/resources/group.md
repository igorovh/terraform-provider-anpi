---
page_title: "anpi_group Resource"
---

# anpi_group

A monitor group: a section of the dashboard and the status page. Deleting a group leaves its monitors ungrouped.

## Example

```hcl
resource "anpi_group" "product" {
  name       = "Product"
  sort_order = 10
}
```

## Argument Reference

- `name` (String, required): up to 60 characters.
- `sort_order` (Number, optional): lower numbers come first. When left out, a new group goes last and an existing group keeps its place.

## Attribute Reference

- `id` (Number): group ID.

## Import

```sh
terraform import anpi_group.product 3
```
