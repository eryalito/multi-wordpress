# Multi‑WordPress on Kubernetes

Deploy and manage multiple WordPress sites on a single Kubernetes deployment, using a lightweight Go controller that provisions WordPress files, configures the internal proxy, and keeps everything in sync.

## Highlights

- Multiple sites in one deployment (one Pod)
- Keep using your own MySQL/MariaDB instances (external or in‑cluster)
- Automatic WordPress setup per domain (files + wp‑config)
- Ingress support for clean domain routing
- Persistent storage for your site files
- Periodic reconciliation to catch config changes

Great for dev, demos, or small environments.

## What you need

- A Kubernetes cluster (1.24+ recommended)
- Helm 3
- A reachable MySQL/MariaDB for each site (host, port, user, password, database)

## Quick start

1) Create a minimal values file (e.g., `my-values.yaml`):

```yaml
ingress:
  enabled: true
  hosts:
    - host: site1.example.com
      paths: ["/"]
    - host: site2.example.com
      paths: ["/"]

storage:
  enabled: true
  class: "standard"
  size: "5Gi"

config:
  wordpress_global:
    zip_url: "https://wordpress.org/latest.zip"
    base_path: "/var/www/html"
  sites:
    - domain_name: "site1.example.com"
      wordpress:
        database:
          host: "db1"
          port: 3306
          user: "wp_user"
          password: "wp_pass"
          name: "wordpress_db1"
    - domain_name: "site2.example.com"
      wordpress:
        database:
          host: "db2"
          port: 3306
          user: "wp_user2"
          password: "wp_pass2"
          name: "wordpress_db2"
```

2) Install the chart from this repo:

```bash
helm install multi-wordpress ./chart/multi-wordpress -f my-values.yaml
```

3) Point DNS for your domains to your cluster’s Ingress and browse to your sites.

## Manage your sites

- Add or remove a site: edit your values file and run `helm upgrade`.
- Update DB credentials: edit values and upgrade; configs are refreshed automatically.
- Storage: increase the requested size in values (if your StorageClass supports expansion) and upgrade.
- HTTPS: use your cluster’s Ingress controller + cert-manager (or any TLS you prefer).

> Note: When using TLS on your ingress use the `force_https` key on each wordpress config.

## Troubleshooting

- Seeing a default/403 page? Make sure the domain is listed under `ingress.hosts` and in `config.sites`.
- Database errors? Verify host/port/user/password/database are correct and reachable from the cluster.
- Changes not applied yet? The reconciler runs periodically. You can also `helm upgrade` to apply immediately.

## Defaults and options

This chart has sensible defaults. Explore all available options in:

- `chart/multi-wordpress/values.yaml`

---

Questions or ideas? Open an issue or PR.
