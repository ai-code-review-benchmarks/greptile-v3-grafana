---
aliases:
  - ../data-sources/prometheus/
  - ../features/datasources/prometheus/
description: Guide for using Prometheus in Grafana
keywords:
  - grafana
  - prometheus
  - guide
labels:
  products:
    - cloud
    - enterprise
    - oss
menuTitle: Prometheus
title: Prometheus data source
weight: 1300
refs:
  build-dashboards:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/dashboards/build-dashboards/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/dashboards/build-dashboards/
  get-started-prometheus:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/getting-started/get-started-grafana-prometheus/#get-started-with-grafana-and-prometheus
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/getting-started/get-started-grafana-prometheus/#get-started-with-grafana-and-prometheus
  configure-grafana-configuration-file-location:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/configure-grafana/#configuration-file-location
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/configure-grafana/#configuration-file-location
  provisioning-data-sources:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/administration/provisioning/#data-sources
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/administration/provisioning/#data-sources
  explore:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/explore/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/explore/
  set-up-grafana-monitoring:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/set-up-grafana-monitoring/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/set-up-grafana-monitoring/
  configure-grafana:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/configure-grafana/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/configure-grafana/
  administration-documentation:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/administration/data-source-management/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/administration/data-source-management/
  annotate-visualizations:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/dashboards/build-dashboards/annotate-visualizations/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana-cloud/visualizations/dashboards/build-dashboards/annotate-visualizations/
  exemplars:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/fundamentals/exemplars/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/fundamentals/exemplars/
  intro-to-prometheus:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/fundamentals/intro-to-prometheus/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/fundamentals/intro-to-prometheus/
  configure-prometheus-data-source:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/datasources/prometheus/configure
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/datasources/prometheus/configure
  azure-active-directory:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/datasources/azure-monitor/#configure-azure-active-directory-ad-authentication
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/datasources/azure-monitor/#configure-azure-active-directory-ad-authentication
---

# Prometheus data source

Prometheus is an open-source database that uses a telemetry collector agent to scrape and store metrics used for monitoring and alerting. 
<!-- 
If you are just getting started with Prometheus, refer to [What is Prometheus?](ref:intro-to-prometheus) -->

Grafana provides native support for Prometheus, so there's no need to install a plugin. 

The following documentation will help you get started working with Prometheus and Grafana:

- [What is Prometheus?](ref:intro-to-prometheus)
- [Prometheus data model](https://prometheus.io/docs/concepts/data_model/)
- [Getting started](https://prometheus.io/docs/prometheus/latest/getting_started/)
- [Configure the Prometheus data source](ref:configure-prometheus-data-source)
- [Prometheus query editor](query-editor/)
- [Template variables](template-variables/)

## Exemplars

In Prometheus, an **exemplar** is a specific trace that represents a measurement taken within a given time interval. While metrics provide an aggregated view of your system, and traces offer a detailed view of individual requests, exemplars serve as a bridge between the two, linking high-level metrics to specific traces for deeper insights. 

Exemplars associate higher-cardinality metadata from a specific event with traditional time series data. Refer to [Introduction to exemplars](ref:exemplars) in Prometheus' documentation for detailed information on how they work.

Grafana can show exemplars data alongside a metric both in Explore and in Dashboards.

{{< figure src="/static/img/docs/v74/exemplars.png" class="docs-image--no-shadow" caption="Exemplar window" >}}

You add exemplars when you configure the Prometheus data source.

{{< figure src="/static/img/docs/prometheus/exemplars-10-1.png" max-width="500px" class="docs-image--no-shadow" >}}

## Prometheus API

The Prometheus data source also works with other projects that implement the [Prometheus querying API](https://prometheus.io/docs/prometheus/latest/querying/api/).

For more information on how to query other Prometheus-compatible projects from Grafana, refer to the specific product's documentation:

- [Grafana Mimir](/docs/mimir/latest/)
- [Thanos](https://thanos.io/tip/components/query.md/)

## View Grafana metrics with Prometheus

Grafana exposes metrics for Prometheus on the `/metrics` endpoint, and comes with a bre-built dashboard to help you start visualizing your metrics right away.

**To import the bundled dashboard:**

1. Navigate to the Prometheus [configuration page](ref:configure-prometheus-data-source).
1. Click the **Dashboards** tab.
1. Locate the **Grafana metrics** dashboard in the list and click **Import**. 

For details about these metrics, refer to [Internal Grafana metrics](ref:set-up-grafana-monitoring).

## Amazon Managed Service for Prometheus

The Prometheus data source with Amazon Managed Service for Prometheus is deprecated. Use the [Amazon Managed service for Prometheus data source](https://grafana.com/grafana/plugins/grafana-amazonprometheus-datasource/) instead. Migration steps are outlined in the linked documentation.

## Azure authentication settings

The Prometheus data source works with Azure authentication. To configure Azure authentication refer to [Configure Azure Active Directory (AD) authentication](ref:azure-active-directory).

In Grafana Enterprise, you need to update the .ini configuration file. Refer to [Configuration file location](ref:configure-grafana-configuration-file-location) to locate your .ini file.

 <!-- Depending on your setup, the .ini file is located [here](ref:configure-grafana-configuration-file-location). -->

Add the following setting in the **[auth]** section :

```bash
[auth]
azure_auth_enabled = true
```

{{% admonition type="note" %}}
If you are using Azure authentication, don't enable `Forward OAuth identity`. Both methods use the same HTTP authorization headers, and the OAuth token will override your Azure credentials.
{{% /admonition %}}

## Get the most out of the Prometheus data source

After installing and configuring Prometheus you can:

- Create a wide variety of [visualizations](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/panels-visualizations/visualizations/)
- Configure and use [templates and variables](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/variables/)
- Add [transformations](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/panels/transformations/)
- Add [annotations](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/dashboards/annotations/)
- Set up [alerting](https://grafana.com/docs/grafana/<GRAFANA_VERSION>/alerting/)
- Create [recorded queries](https://grafana.com/docs/grafana/latest/administration/recorded-queries/)
