# Grafana Dashboard Render Performance Metrics

This documentation describes the dashboard load performance metrics exposed from Grafana's frontend.

## Overview

The exposed dashboard performance metrics feature provides comprehensive tracking and profiling of dashboard interactions, allowing administrators and developers to analyze dashboard render performance, user interactions, and identify performance bottlenecks.

## Configuration

### Enabling Performance Metrics

Dashboard performance metrics are configured in the Grafana configuration file (`grafana.ini`) under the `[dashboards]` section:

```ini
[dashboards]
# Dashboards UIDs to report performance metrics for. * can be used to report metrics for all dashboards
dashboard_performance_metrics = *
```

**Configuration Options:**

- **`*`** - Enable profiling on all dashboards
- **`<comma-separated-list-of-dashboard-uid>`** - Enable profiling on specific dashboards only
- **`""` (empty)** - Disable performance metrics (default)

**Examples:**

```ini
# Enable for all dashboards
dashboard_performance_metrics = *

# Enable for specific dashboards
dashboard_performance_metrics = dashboard-uid-1,dashboard-uid-2,dashboard-uid-3

# Disable performance metrics
dashboard_performance_metrics =
```

## Tracked Interactions

The system tracks various dashboard interaction types automatically using the [`@grafana/scenes`](https://github.com/grafana/scenes) library. Each interaction is captured with a specific origin identifier that describes the type of user action performed.

### Core Performance-Tracked Interactions

The following dashboard interaction types are tracked for dashboard render performance profiling:

| Interaction Type         | Trigger                    | Description                           | When Measured                                            |
| ------------------------ | -------------------------- | ------------------------------------- | -------------------------------------------------------- |
| `dashboard_view`         | Initial dashboard view     | Initial dashboard rendering           | When user first loads or navigates to a dashboard        |
| `refresh`                | Manual/Auto refresh        | Dashboard refresh operations          | When user clicks refresh button or auto-refresh triggers |
| `time_range_change`      | Time picker changes        | Time range modifications              | When user changes time range in time picker              |
| `filter_added`           | Ad-hoc filter addition     | Adding new ad-hoc filters             | When user adds a new filter to the dashboard             |
| `filter_removed`         | Ad-hoc filter removal      | Removing existing ad-hoc filters      | When user removes a filter from the dashboard            |
| `filter_changes`         | Ad-hoc filter modification | Modifying existing ad-hoc filters     | When user changes filter values or operators             |
| `filter_restored`        | Ad-hoc filter restoration  | Restoring previously removed filters  | When user restores a previously applied filter           |
| `variable_value_changed` | Variable value changes     | Template variable value modifications | When user changes dashboard variable values              |
| `scopes_changed`         | Scopes modifications       | Dashboard scopes changes              | When user modifies dashboard scopes                      |

The interactions mentioned above are reported to Echo service as well as sent to [Faro](https://grafana.com/docs/grafana-cloud/monitor-applications/frontend-observability/) as `dashboard_render` measurements:

```ts
const payload = {
  duration: e.duration,
  networkDuration: e.networkDuration,
  totalJSHeapSize: e.totalJSHeapSize,
  usedJSHeapSize: e.usedJSHeapSize,
  jsHeapSizeLimit: e.jsHeapSizeLimit,
  timeSinceBoot: performance.measure('time_since_boot', 'frontend_boot_js_done_time_seconds').duration,
};

reportInteraction('dashboard_render', {
  interactionType: e.origin,
  uid,
  ...payload,
});

logMeasurement(`dashboard_render`, payload, { interactionType: e.origin, dashboard: uid, title: title });
```

### Interaction Origin Mapping

The profiling system uses profiler event's `origin` directly as the `interactionType`, providing direct mapping between user actions and performance measurements.

## Profiling Implementation

### Profile Data Structure

Each interaction profile event captures:

```typescript
interface SceneInteractionProfileEvent {
  origin: string; // Interaction type
  duration: number; // Total interaction duration
  networkDuration: number; // Network requests duration
  totalJSHeapSize: number; // JavaScript heap size metrics
  usedJSHeapSize: number; // Used JavaScript heap size
  jsHeapSizeLimit: number; // JavaScript heap size limit
}
```

### Collected Metrics

For each tracked interaction, the system collects:

- **Dashboard Metadata**: UID, title
- **Performance Metrics**: Duration, network duration
- **Memory Metrics**: JavaScript heap usage statistics
- **Timing Information**: Time since boot
- **Interaction Context**: Type of user interaction

## Debugging and Development

### Enable Debug Logging

To observe profiling events in the browser console:

```javascript
localStorage.setItem('grafana.debug.scenes', 'true');
```

### Console Output

When debug logging is enabled, you'll see console logs for each profiling event:

```
SceneRenderProfiler: Profile started: {origin: <NAME_OF_INTERACTION>, crumbs: Array(0)}
... // intermediate steps adding profile crumbs
SceneRenderProfiler: Stopped recording, total measured time (network included): 2123
```

### Browser Performance Profiler

Dashboard interactions can be recorded in the browser's performance profiler, where they appear as:

```
Dashboard Interaction <NAME_OF_INTERACTION>
```

## Analytics Integration

### Interaction Reporting

Performance data is integrated with Grafana's analytics system through:

- **`reportInteraction`**: Reports interaction events to Echo service with performance data
- **`logMeasurement`**: Records Faro's performance measurements with metadata

### Data Collection

The system reports the following data for each interaction:

```typescript
{
  interactionType: string,      // Type of interaction
  uid: string,                  // Dashboard UID
  duration: number,             // Total duration
  networkDuration: number,      // Network time
  totalJSHeapSize: number,      // Memory metrics
  usedJSHeapSize: number,
  jsHeapSizeLimit: number,
  timeSinceBoot: number         // Time since frontend boot
}
```

## Implementation Details

The profiler is integrated into dashboard creation paths and uses a singleton pattern to share profiler instances across dashboard reloads. The performance tracking is implemented using the `SceneRenderProfiler` from the `@grafana/scenes` library.

## Related Documentation

- [PR #858 - Add SceneRenderProfiler to scenes](https://github.com/grafana/scenes/pull/858)
- [PR #99629 - Dashboard render performance metrics](https://github.com/grafana/grafana/pull/99629)
- [PR #108658 - Dashboard: Tweak interaction tracking](https://github.com/grafana/grafana/pull/108658)
- [PR #1195 - Enhance SceneRenderProfiler with additional interaction tracking](https://github.com/grafana/scenes/pull/1195)
- [PR #1198 - Make SceneRenderProfiler optional and injectable](https://github.com/grafana/scenes/pull/1198)
