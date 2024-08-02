---
aliases:
  - ../administration/reports/
  - ../enterprise/export-pdf/
  - ../enterprise/reporting/
  - ../reference/share_dashboard/
  - ../reference/share_panel/
  - ../share-dashboards-panels/
  - ../sharing/
  - ../sharing/playlists/
  - ../sharing/share-dashboard/
  - ../sharing/share-panel/
  - ./
  - reporting/
  - share-dashboard/
keywords:
  - grafana
  - dashboard
  - documentation
  - share
  - panel
  - library panel
  - playlist
  - reporting
  - export
  - pdf
labels:
  products:
    - cloud
    - enterprise
    - oss
menuTitle: Sharing
title: Share dashboards and panels
description: Share Grafana dashboards and panels within your organization and publicly
weight: 85
refs:
  image-rendering:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/image-rendering/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/setup-grafana/image-rendering/
  grafana-enterprise:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/introduction/grafana-enterprise/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/introduction/grafana-enterprise/
  shared-dashboards:
    - pattern: /docs/grafana/
      destination: /docs/grafana/<GRAFANA_VERSION>/dashboards/share-dashboards-panels/shared-dashboards/
    - pattern: /docs/grafana-cloud/
      destination: /docs/grafana/<GRAFANA_VERSION>/visualizations/dashboards/share-dashboards-panels/shared-dashboards/
---

# Share dashboards and panels

Grafana enables you to share dashboards and panels with other users within an organization and in certain situations, publicly on the Web. You can share using:

- Direct links with users in and outside of your organization
- Snapshots
- Embeds
- PDFs
- JSON files
- Reports

You must have an authorized viewer permission to see an image rendered by a direct link.

The same permission is also required to view embedded links unless you have anonymous access permission enabled for your Grafana instance.

{{< admonition type="note" >}}
As of Grafana 8.0, anonymous access permission is not available in Grafana Cloud.
{{< /admonition >}}

## Share a dashboard

You can share dashboards in the following ways:

- [Internally with a link](#share-internally)
- [Externally with anyone or specific people](#share-externally)
- [As a report](#schedule-a-report)
- [As a snapshot](#share-a-snapshot)
- [As a PDF export](#export-a-dashboard-as-pdf)
- [As a JSON file export](#export-a-dashboard-as-json)

When you share a dashboard externally as a link or by email, those dashboards are included a list of your shared dashboards. To view the list and manage these dashboards, navigate to **Dashboards > Shared dashboards**.

<!-- image of list here -->

### Share internally

Share a personalized, direct link to your dashboard within your organization.

1. Click **Dashboards** in the main menu.
1. Click the dashboard you want to share.
1. Click the **Share** drop-down in the top-right corner and select **Share internally**.
1. (Optional) Set the following options (they're enabled by default):
   - **Lock time range** - Change the current relative time range to an absolute time range.
   - **Shorten link** - Shorten the dashboard link.
1. Select the **Current**, **Dark**, or **Light** theme for the dashboard.
1. Click **Copy link**.
1. Send the copied link to a Grafana user with authorization to view the link.
1. Click the **X** at the top-right corner to close the share drawer.

Once you have a customized internal link, you can share it quickly by following these steps:

1. Click **Dashboards** in the main menu.
1. Click the dashboard you want to share.
1. Click the **Share** button to copy a shortened link.

This link has any customizations like time range locking or theme you've previously set.

### Share externally

Externally shared dashboards allow you to share your Grafana dashboard with anyone. This is useful when you want to make your dashboard available to the world without requiring access to your Grafana organization.

Learn how to configure and manage externally shared dashboards in [Externally shared dashboards](ref:shared-dashboards).

### Schedule a report

{{< admonition type="note" >}}
This feature is only available in Grafana Enterprise.
{{< /admonition >}}

To share your dashboard as a report, follow these steps:

1. Click **Dashboards** in the main menu.
1. Click the dashboard you want to share.
1. Click the **Share** drop-down in the top-right corner and select **Schedule a report**.
1. [Configure the report](ref:configure-report).
1. Depending on your schedule settings, click **Schedule send** or **Send now**.

You can also save the report as a draft.

To manage your reports, navigate to **Dashboards > Reporting > Reports**.

### Share a snapshot

A dashboard snapshot publicly shares a dashboard while removing sensitive data such as queries and panel links, leaving only visible metrics and series names. Anyone with the link can access the snapshot.

You can publish snapshots to your local instance or to [snapshots.raintank.io](http://snapshots.raintank.io). The latter is a free service provided by Grafana Labs that enables you to publish dashboard snapshots to an external Grafana instance. Anyone with the link can view it. You can set an expiration time if you want the snapshot removed after a certain time period.

{{< admonition type=note >}}
The snapshots.raintank.io option is disabled by default in Grafana Cloud. To enable it...
{{< /admonition >}}

To share your dashboard with anyone as a snapshot, follow these steps.

1. Click **Dashboards** in the main menu.
1. Click the dashboard you want to share.
1. Click the **Share** drop-down in the top-right corner and select **Share snapshot**.
1. In the **Snapshot name** field, enter a descriptive title for the snapshot.
1. Select one of the following expiration options for the snapshot:
   - **1 Hour**
   - **1 Day**
   - **1 Week**
   - **Never**
1. Click **Publish snapshot**.
1. (Optional) If you want to see the other snapshots shared from your organization, click the **View all snapshots** link.

   You can also navigate to **Dashboards > Snapshots** in the primary menu.

1. Click the **X** at the top-right corner to close the share drawer.

#### Delete a snapshot

To delete existing snapshots, follow these steps:

1. Navigate to **Dashboards > Snapshots** in the main menu.
1. Click the red **x** next to the snapshot that you want to delete.

The snapshot is immediately deleted. You may need to clear your browser cache or use a private or incognito browser to confirm this.

## Export a dashboard

In addition to sharing dashboards as links, reports, and snapshots, you can export them as PDFs or JSON files.

### Export a dashboard as PDF

To export a dashboard in its current state as a PDF, follow these steps:

1. Click **Dashboards** in the main menu.
1. Open the dashboard you want to export.
1. Click the **Export** drop-down in the top-right corner and select **Export as PDF**.
1. Select either **Landscape** or **Portrait** for the PDF orientation.
1. Select either **Grid** or **Simple** for the PDF layout.
1. Set the **Zoom** level, which increases or decreases the numbrer of rows and columns in table visualizations.
1. Click **Generate PDF**.

   The PDF opens in another tab where you can download it.

1. Click the **X** at the top-right corner to close the share drawer.

### Export a dashboard as JSON

Export a Grafana JSON file that contains everything you need, including layout, variables, styles, data sources, queries, and so on, so that you can later import the dashboard. To export a JSON file, follow these steps:

1. Click **Dashboards** in the main menu.
1. Open the dashboard you want to export.
1. Click the **Export** drop-down in the top-right corner and select **Export as JSON**.
1. If you're exporting the dashboard to use in another instance, with different data source UIDs, enable the **Export for sharing externally** switch.
1. Click **Download file** or **Copy to clipboard**.
1. Click the **X** at the top-right corner to close the share drawer.

## Share a panel

You can share a panel as a direct link, as a snapshot, or as an embedded link. You can also create library panels using the **Share** option on any panel.

1. Hover over any part of the panel to display the actions menu on the top right corner.
1. Click the menu and select **Share**.

   The share dialog opens and shows the **Link** tab.

### Use direct link

The **Link** tab shows the current time range, template variables, and the default theme. You can optionally enable a shortened URL to share.

1. Click **Copy**.

   This action copies the default or the shortened URL to the clipboard.

1. Send the copied URL to a Grafana user with authorization to view the link.
1. You also optionally click **Direct link rendered image** to share an image of the panel.

For more information, refer to [Image rendering](ref:image-rendering).

The following example shows a link to a server-side rendered PNG:

```bash
https://play.grafana.org/d/000000012/grafana-play-home?orgId=1&from=1568719680173&to=1568726880174&panelId=4&fullscreen
```

#### Query string parameters for server-side rendered images

- **width:** Width in pixels. Default is 800.
- **height:** Height in pixels. Default is 400.
- **tz:** Timezone in the format `UTC%2BHH%3AMM` where HH and MM are offset in hours and minutes after UTC
- **timeout:** Number of seconds. The timeout can be increased if the query for the panel needs more than the default 30 seconds.
- **scale:** Numeric value to configure device scale factor. Default is 1. Use a higher value to produce more detailed images (higher DPI). Supported in Grafana v7.0+.

### Publish a snapshot

A panel snapshot shares an interactive panel publicly. Grafana strips sensitive data leaving only the visible metric data and series names embedded in the dashboard. Panel snapshots can be accessed by anyone with the link.

You can publish snapshots to your local instance or to [snapshots.raintank.io](http://snapshots.raintank.io). The latter is a free service provided by [Grafana Labs](https://grafana.com), that enables you to publish dashboard snapshots to an external Grafana instance.

{{< admonition type="note" >}}
As of Grafana 11, the option to publish to [snapshots.raintank.io](http://snapshots.raintank.io) is no longer available for Grafana Cloud.
{{< /admonition >}}

You can optionally set an expiration time if you want the snapshot to be removed after a certain time period.

1. In the **Share Panel** dialog, click **Snapshot** to go to the tab.
1. Click **Publish to snapshots.raintank.io** or **Publish Snapshot**.

   Grafana generates the link of the snapshot.

1. Copy the snapshot link, and share it either within your organization or publicly on the web.

If you created a snapshot by mistake, click **Delete snapshot** in the dialog box to remove the snapshot from your Grafana instance.

#### Delete a snapshot

To delete existing snapshots, follow these steps:

1. Click **Dashboards** in the main menu.
1. Click **Snapshots** to go to the snapshots management page.
1. Click the red **x** next to the snapshot URL that you want to delete.

The snapshot is immediately deleted. You may need to clear your browser cache or use a private or incognito browser to confirm this.

### Embed panel

You can embed a panel using an iframe on another web site. A viewer must be signed into Grafana to view the graph.

{{< admonition type="note" >}}
As of Grafana 8.0, anonymous access permission is no longer available for Grafana Cloud.
{{< /admonition >}}

Here is an example of the HTML code:

```html
<iframe
  src="https://snapshots.raintank.io/dashboard-solo/snapshot/y7zwi2bZ7FcoTlB93WN7yWO4aMiz3pZb?from=1493369923321&to=1493377123321&panelId=4"
  width="650"
  height="300"
  frameborder="0"
></iframe>
```

The result is an interactive Grafana graph embedded in an iframe.

### Library panel

To create a library panel from the **Share Panel** dialog:

1. Click **Library panel**.
1. In **Library panel name**, enter the name.
1. In **Save in folder**, select the folder in which to save the library panel. By default, the root level is selected.
1. Click **Create library panel** to save your changes.
1. Click **Save dashboard**.
