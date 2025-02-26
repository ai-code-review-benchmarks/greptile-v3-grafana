import { css, cx } from '@emotion/css';
import { useMemo } from 'react';

import { GrafanaTheme2, textUtil } from '@grafana/data';
import { VizPanel } from '@grafana/scenes';
import { useStyles2, Text, Icon, Stack, Tooltip } from '@grafana/ui';
import { t } from 'app/core/internationalization';
import { OptionsPaneCategoryDescriptor } from 'app/features/dashboard/components/PanelEditor/OptionsPaneCategoryDescriptor';
import { OptionsPaneItemDescriptor } from 'app/features/dashboard/components/PanelEditor/OptionsPaneItemDescriptor';

import {
  PanelBackgroundSwitch,
  PanelDescriptionTextArea,
  PanelFrameTitleInput,
} from '../panel-edit/getPanelFrameOptions';
import { BulkActionElement } from '../scene/types/BulkActionElement';
import { isDashboardLayoutItem } from '../scene/types/DashboardLayoutItem';
import { EditableDashboardElement } from '../scene/types/EditableDashboardElement';
import { dashboardSceneGraph } from '../utils/dashboardSceneGraph';
import { getEditPanelUrl } from '../utils/urlBuilders';
import { getPanelIdForVizPanel } from '../utils/utils';

import { renderTitle } from './shared';

export class VizPanelEditableElement implements EditableDashboardElement, BulkActionElement {
  public readonly isEditableDashboardElement = true;
  public readonly typeName = 'Panel';
  public readonly alwaysExpanded = true;

  public constructor(private panel: VizPanel) {}

  public getPanel = () => {
    return this.panel;
  };

  public useEditPaneOptions(): OptionsPaneCategoryDescriptor[] {
    const panel = this.panel;
    const layoutElement = panel.parent!;

    const panelOptions = useMemo(() => {
      return new OptionsPaneCategoryDescriptor({
        title: ``,
        id: 'panel-options',
        isOpenDefault: true,
        alwaysExpanded: true,
        renderTitle: () => renderTitle({ title: 'Panel', onDelete: this.onDelete }),
      })
        .addItem(
          new OptionsPaneItemDescriptor({
            title: '',
            render: () => <OpenPanelEditViz model={this} />,
          })
        )
        .addItem(
          new OptionsPaneItemDescriptor({
            title: t('dashboard.viz-panel.options.title-option', 'Title'),
            value: panel.state.title,
            popularRank: 1,
            render: function renderTitle() {
              return <PanelFrameTitleInput panel={panel} />;
            },
          })
        )
        .addItem(
          new OptionsPaneItemDescriptor({
            title: t('dashboard.viz-panel.options.description', 'Description'),
            value: panel.state.description,
            render: function renderDescription() {
              return <PanelDescriptionTextArea panel={panel} />;
            },
          })
        )
        .addItem(
          new OptionsPaneItemDescriptor({
            title: t('dashboard.viz-panel.options.transparent-background', 'Transparent background'),
            render: function renderTransparent() {
              return <PanelBackgroundSwitch panel={panel} />;
            },
          })
        );
    }, [panel]);

    const layoutCategory = useMemo(() => {
      if (isDashboardLayoutItem(layoutElement) && layoutElement.getOptions) {
        return layoutElement.getOptions();
      }
      return undefined;
    }, [layoutElement]);

    const categories = [panelOptions];
    if (layoutCategory) {
      categories.push(layoutCategory);
    }

    return categories;
  }

  public onDelete = () => {
    const layout = dashboardSceneGraph.getLayoutManagerFor(this.panel);
    layout.removePanel?.(this.panel);
  };
}

const getStyles = (theme: GrafanaTheme2) => ({
  pluginDescriptionWrapper: css({
    display: 'flex',
    flexWrap: 'nowrap',
    alignItems: 'center',
    columnGap: theme.spacing(1),
    rowGap: theme.spacing(0.5),
    minHeight: theme.spacing(4),
    backgroundColor: theme.components.input.background,
    border: `1px solid ${theme.colors.border.strong}`,
    borderRadius: theme.shape.radius.default,
    paddingInline: theme.spacing(1),
    paddingBlock: theme.spacing(0.5),
    flexGrow: 1,
  }),
  panelVizImg: css({
    width: '16px',
    height: '16px',
    marginRight: theme.spacing(1),
  }),
  panelVizIcon: css({
    marginLeft: 'auto',
  }),
});

type OpenPanelEditVizProps = {
  model: VizPanelEditableElement;
};

const OpenPanelEditViz = ({ model }: OpenPanelEditVizProps) => {
  const styles = useStyles2(getStyles);

  const plugin = model.getPanel().getPlugin();
  const imgSrc = plugin?.meta.info.logos.small;

  return (
    <>
      <Stack alignItems="center" width="100%">
        {plugin ? (
          <Tooltip content="Open Panel Edit">
            <a
              href={textUtil.sanitizeUrl(getEditPanelUrl(getPanelIdForVizPanel(model.getPanel())))}
              className={cx(styles.pluginDescriptionWrapper)}
              onClick={() => {}}
            >
              <img className={styles.panelVizImg} src={imgSrc} alt="Image of plugin type" />
              <Text truncate>{plugin.meta.name}</Text>
              <Icon className={styles.panelVizIcon} name="sliders-v-alt" />
            </a>
          </Tooltip>
        ) : null}
      </Stack>
    </>
  );
};
