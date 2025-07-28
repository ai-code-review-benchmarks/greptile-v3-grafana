// Core Grafana history https://github.com/grafana/grafana/blob/v11.0.0-preview/public/app/plugins/datasource/prometheus/querybuilder/components/metrics-modal/styles.ts
import { css } from '@emotion/css';

import { GrafanaTheme2 } from '@grafana/data';

export const getStyles = (theme: GrafanaTheme2, disableTextWrap: boolean) => {
  return {
    modal: css({
      width: '85vw',
      [theme.breakpoints.down('md')]: {
        width: '100%',
      },
      [theme.breakpoints.up('xl')]: {
        width: '60%',
      },
    }),
    inputWrapper: css({
      display: 'flex',
      flexDirection: 'row',
      flexWrap: 'wrap',
    }),
    inputItemFirst: css({
      flexBasis: '40%',
      paddingRight: '16px',
      [theme.breakpoints.down('md')]: {
        paddingRight: '0px',
        paddingBottom: '16px',
      },
    }),
    inputItem: css({
      flexGrow: 1,
      flexBasis: '20%',
      [theme.breakpoints.down('md')]: {
        minWidth: '100%',
      },
    }),
    resultsData: css({
      margin: `4px 0 ${theme.spacing(2)} 0`,
    }),
    resultsDataFiltered: css({
      color: theme.colors.text.secondary,
      textAlign: 'center',
      border: 'solid 1px rgba(204, 204, 220, 0.25)',
      padding: '7px',
    }),
    resultsDataFilteredText: css({
      display: 'inline',
      verticalAlign: 'text-top',
    }),
    results: css({
      height: 'calc(80vh - 310px)',
      overflowY: 'scroll',
    }),
    resultsFooter: css({
      marginTop: '24px',
      display: 'flex',
      flexDirection: 'row',
      flexWrap: 'wrap',
      alignItems: 'center',
      position: 'sticky',
      justifyContent: 'center',
    }),
    currentlySelected: css({
      color: 'grey',
      opacity: '75%',
      fontSize: '0.75rem',
    }),
    loadingSpinner: css({
      visibility: 'hidden',
    }),
    visible: css({
      visibility: 'visible',
    }),
    noBorder: css({
      border: 'none',
    }),
  };
};
