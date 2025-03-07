import { css } from '@emotion/css';
import { useRef } from 'react';
import { v4 as uuidv4 } from 'uuid';

import { GrafanaTheme2 } from '@grafana/data';
import { config } from 'app/core/config';
import { DimensionContext } from 'app/features/dimensions';
import { ColorDimensionEditor } from 'app/features/dimensions/editors/ColorDimensionEditor';
import { TextDimensionEditor } from 'app/features/dimensions/editors/TextDimensionEditor';

import {
  CanvasElementItem,
  CanvasElementProps,
  CanvasElementOptions,
  defaultBgColor,
  defaultTextColor,
} from '../element';
import { Align, CanvasElementConfig, CanvasElementData, VAlign } from '../types';

const Player = (props: CanvasElementProps<CanvasElementConfig, CanvasElementData>) => {
  const { data } = props;
  const styles = getStyles(config.theme2, data);
  const uniqueId = useRef(uuidv4());

  return (
    <div className={styles.container}>
      <svg
        xmlns="http://www.w3.org/2000/svg"
        viewBox="0 0 200 200"
        width="100%"
        height="100%"
        preserveAspectRatio="none"
      >
        <defs>
          <pattern id={`image-${uniqueId.current}`} patternUnits="userSpaceOnUse" width="200" height="200">
            <image xlinkHref={data?.backgroundImage} x="-50" y="-50" width="300" height="300"></image>
          </pattern>
          <clipPath id={`triangleClip-${uniqueId.current}`}>
            <polygon points="100,5 195,195 5,195" />
          </clipPath>
          <linearGradient id={`borderGradient-${uniqueId.current}`} x1="0%" y1="0%" x2="100%" y2="100%">
            <stop offset="0%" stopColor="red">
              <animate
                attributeName="stop-color"
                values="red; blue; green; yellow; red"
                dur="3s"
                repeatCount="indefinite"
              />
            </stop>
            <stop offset="50%" stopColor="blue">
              <animate
                attributeName="stop-color"
                values="blue; green; yellow; red; blue"
                dur="3s"
                repeatCount="indefinite"
              />
            </stop>
            <stop offset="100%" stopColor="green">
              <animate
                attributeName="stop-color"
                values="green; yellow; red; blue; green"
                dur="3s"
                repeatCount="indefinite"
              />
            </stop>
          </linearGradient>
        </defs>

        {/* Apply background image within the clipping area */}
        <rect
          x="0"
          y="0"
          width="100%"
          height="100%"
          clipPath={`url(#triangleClip-${uniqueId.current})`}
          style={{ fill: 'none' }}
        />
        <polygon
          points="100,5 195,195 5,195"
          style={{ fill: data?.backgroundImage ? `url(#image-${uniqueId.current})` : data?.backgroundColor }}
        />

        {/* Animated Border with Wider Stroke */}
        <polygon
          points="100,5 195,195 5,195"
          fill="none"
          stroke={`url(#borderGradient-${uniqueId.current})`}
          strokeWidth="24"
          strokeLinejoin="round"
          strokeLinecap="round"
          strokeDasharray="0"
        />
      </svg>

      <span className={styles.text}>{data?.text}</span>
    </div>
  );
};

export const playerItem: CanvasElementItem = {
  id: 'player',
  name: 'Player',
  description: 'Player',

  display: Player,

  defaultSize: {
    width: 24,
    height: 24,
  },

  getNewOptions: (options) => ({
    ...options,
    background: {
      color: {
        fixed: defaultBgColor,
      },
    },
    config: {
      align: Align.Center,
      valign: VAlign.Middle,
      color: {
        fixed: defaultTextColor,
      },
    },
    placement: {
      width: options?.placement?.width ?? 160,
      height: options?.placement?.height ?? 138,
      top: options?.placement?.top,
      left: options?.placement?.left,
      rotation: options?.placement?.rotation ?? 0,
    },
    links: options?.links ?? [],
  }),

  // Called when data changes
  prepareData: (dimensionContext: DimensionContext, elementOptions: CanvasElementOptions<CanvasElementConfig>) => {
    const textConfig = elementOptions.config;

    const data: CanvasElementData = {
      text: textConfig?.text ? dimensionContext.getText(textConfig.text).value() : '',
      field: textConfig?.text?.field,
      align: textConfig?.align ?? Align.Center,
      valign: textConfig?.valign ?? VAlign.Middle,
      size: textConfig?.size,
    };

    if (textConfig?.color) {
      data.color = dimensionContext.getColor(textConfig.color).value();
    }

    const { background, border } = elementOptions;
    data.backgroundColor = background?.color ? dimensionContext.getColor(background.color).value() : defaultBgColor;
    data.borderColor = border?.color ? dimensionContext.getColor(border.color).value() : defaultBgColor;
    data.borderWidth = border?.width ?? 0;

    data.backgroundImage = background?.image ? dimensionContext.getResource(background.image).value() : undefined;

    return data;
  },

  registerOptionsUI: (builder) => {
    const category = ['Player'];
    builder
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

  customConnectionAnchors: [
    // points along the left edge
    { x: -1, y: -1 }, // bottom left
    { x: -0.8, y: -0.6 },
    { x: -0.6, y: -0.2 },
    { x: -0.4, y: 0.2 },
    { x: -0.2, y: 0.6 },
    { x: 0, y: 1 }, // top

    // points along the right edge
    { x: 0.2, y: 0.6 },
    { x: 0.4, y: 0.2 },
    { x: 0.6, y: -0.2 },
    { x: 0.8, y: -0.6 },
    { x: 1, y: -1 }, // bottom right

    // points along the bottom edge
    { x: 0.6, y: -1 },
    { x: 0.2, y: -1 },
    { x: -0.2, y: -1 },
    { x: -0.6, y: -1 },
  ],
};

const getStyles = (theme: GrafanaTheme2, data: CanvasElementData | undefined) => {
  const textTop = data?.valign === VAlign.Middle ? '50%' : data?.valign === VAlign.Top ? '10%' : '90%';
  const textLeft = data?.align === Align.Center ? '50%' : data?.align === Align.Left ? '10%' : '90%';
  const textTransform = `translate(${data?.align === Align.Center ? '-50%' : data?.align === Align.Left ? '10%' : '-90%'}, ${
    data?.valign === VAlign.Middle ? '-50%' : data?.valign === VAlign.Top ? '10%' : '-90%'
  })`;

  return {
    container: css({
      height: '100%',
      width: '100%',
    }),
    text: css({
      position: 'absolute',
      top: textTop,
      left: textLeft,
      transform: textTransform,
      fontSize: `${data?.size}px`,
      color: data?.color,
    }),
    elementBorder: css({
      fill: 'none',
      stroke: data?.borderColor ?? 'none',
      strokeWidth: data?.borderWidth ?? 0,
      strokeLinejoin: 'round',
    }),
  };
};
