import { css } from '@emotion/css';

import { GrafanaTheme2, SelectableValue } from '@grafana/data';
import { DataFrame, Field } from '@grafana/data/';
import { EmbeddedScene, PanelBuilders, SceneDataNode, SceneFlexItem, SceneFlexLayout } from '@grafana/scenes';
import { stylesFactory } from '@grafana/ui';
import { config } from 'app/core/config';
import { DimensionContext } from 'app/features/dimensions/context';
import { ColorDimensionEditor } from 'app/features/dimensions/editors/ColorDimensionEditor';
import { TextDimensionEditor } from 'app/features/dimensions/editors/TextDimensionEditor';

import {
  CanvasElementItem,
  CanvasElementOptions,
  CanvasElementProps,
  defaultBgColor,
  defaultTextColor,
} from '../element';
import { Align, VAlign, VizElementConfig, VizElementData } from '../types';

const panelTypes: Array<SelectableValue<string>> = Object.keys(PanelBuilders).map((type) => {
  return { label: type, value: type };
});

const VisualizationDisplay = (props: CanvasElementProps<VizElementConfig, VizElementData>) => {
  const { data } = props;
  const styles = getStyles(config.theme2, data);

  let panelToEmbed = PanelBuilders.timeseries().setTitle('Embedded Panel');
  if (data?.vizType) {
    // TODO make this better
    panelToEmbed = PanelBuilders[data.vizType as keyof typeof PanelBuilders]().setTitle('Embedded Panel');
  }

  panelToEmbed.setData(new SceneDataNode({ data: data!.data }));
  const panel = panelToEmbed.build();

  const embeddedPanel = new EmbeddedScene({
    body: new SceneFlexLayout({
      children: [
        new SceneFlexItem({
          width: '100%',
          height: '100%',
          body: panel,
        }),
      ],
    }),
  });

  return (
    <div className={styles.container}>
      <span className={styles.span}>
        <embeddedPanel.Component model={embeddedPanel} />
      </span>
    </div>
  );
};

const getStyles = stylesFactory((theme: GrafanaTheme2, data) => ({
  container: css({
    position: 'absolute',
    height: '100%',
    width: '100%',
    display: 'table',
  }),
  span: css({
    display: 'table-cell',
    verticalAlign: data?.valign,
    textAlign: data?.align,
    fontSize: `${data?.size}px`,
    color: data?.color,
  }),
}));

export const visualizationItem: CanvasElementItem<VizElementConfig, VizElementData> = {
  id: 'visualization',
  name: 'Visualization',
  description: 'Visualization',

  display: VisualizationDisplay,

  defaultSize: {
    width: 240,
    height: 160,
  },

  getNewOptions: (options) => ({
    ...options,
    config: {
      align: Align.Center,
      valign: VAlign.Middle,
      color: {
        fixed: defaultTextColor,
      },
      vizType: 'timeseries',
      fields: options?.fields ?? [],
    },
    background: {
      color: {
        fixed: defaultBgColor,
      },
    },
    links: options?.links ?? [],
  }),

  // Called when data changes
  prepareData: (dimensionContext: DimensionContext, elementOptions: CanvasElementOptions<VizElementConfig>) => {
    const vizConfig = elementOptions.config;
    let panelData = dimensionContext.getPanelData();

    const getMatchingFields = (frame: DataFrame) => {
      let fields: Field[] = [];
      frame.fields.forEach((field) => {
        if (field.type === 'time' || vizConfig?.fields?.includes(field.name)) {
          fields.push(field);
        }
      });

      return fields;
    };

    if (vizConfig?.fields && vizConfig.fields.length > 1 && panelData) {
      let frames = panelData?.series;
      let selectedFrames =
        frames?.filter((frame) => frame.fields.filter((field) => vizConfig.fields!.includes(field.name)).length > 0) ??
        [];

      selectedFrames = selectedFrames?.map((frame) => ({
        ...frame,
        fields: getMatchingFields(frame),
      }));

      panelData = {
        ...panelData,
        series: selectedFrames,
      };
    }

    const data: VizElementData = {
      text: vizConfig?.text ? dimensionContext.getText(vizConfig.text).value() : '',
      field: vizConfig?.text?.field,
      align: vizConfig?.align ?? Align.Center,
      valign: vizConfig?.valign ?? VAlign.Middle,
      size: vizConfig?.size,
      vizType: vizConfig?.vizType,
      data: panelData,
    };

    if (vizConfig?.color) {
      data.color = dimensionContext.getColor(vizConfig.color).value();
    }

    return data;
  },

  // Heatmap overlay options
  registerOptionsUI: (builder) => {
    const category = ['Visualization'];
    builder
      .addSelect({
        category,
        path: 'config.vizType',
        name: 'Viz Type',
        settings: {
          options: panelTypes,
        },
      })
      .addCustomEditor({
        category,
        id: 'textSelector',
        path: 'config.text',
        name: 'Text',
        editor: TextDimensionEditor,
      })
      .addCustomEditor({
        category,
        id: 'config.color',
        path: 'config.color',
        name: 'Text color',
        editor: ColorDimensionEditor,
        settings: {},
        defaultValue: {},
      })
      .addRadio({
        category,
        path: 'config.align',
        name: 'Align text',
        settings: {
          options: [
            { value: Align.Left, label: 'Left' },
            { value: Align.Center, label: 'Center' },
            { value: Align.Right, label: 'Right' },
          ],
        },
        defaultValue: Align.Left,
      })
      .addRadio({
        category,
        path: 'config.valign',
        name: 'Vertical align',
        settings: {
          options: [
            { value: VAlign.Top, label: 'Top' },
            { value: VAlign.Middle, label: 'Middle' },
            { value: VAlign.Bottom, label: 'Bottom' },
          ],
        },
        defaultValue: VAlign.Middle,
      })
      .addNumberInput({
        category,
        path: 'config.size',
        name: 'Text size',
        settings: {
          placeholder: 'Auto',
        },
      });
  },
};
