import { PureComponent } from 'react';

import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { ConnectionConfig } from '@grafana/google-sdk';
import { ConfigSection, DataSourceDescription } from '@grafana/plugin-ui';
import { reportInteraction, config } from '@grafana/runtime';
import { Divider, SecureSocksProxySettings } from '@grafana/ui';

import { CloudMonitoringOptions, CloudMonitoringSecureJsonData } from '../../types/types';

export type Props = DataSourcePluginOptionsEditorProps<CloudMonitoringOptions, CloudMonitoringSecureJsonData>;

export class ConfigEditor extends PureComponent<Props> {
  handleOnOptionsChange = (options: Props['options']) => {
    if (options.jsonData.privateKeyPath || options.secureJsonFields['privateKey']) {
      reportInteraction('grafana_cloud_monitoring_config_changed', {
        authenticationType: 'JWT',
        privateKey: options.secureJsonFields['privateKey'],
        privateKeyPath: !!options.jsonData.privateKeyPath,
      });
    }
    this.props.onOptionsChange(options);
  };

  render() {
    const { options, onOptionsChange } = this.props;
    return (
      <>
        <DataSourceDescription
          dataSourceName="Google Cloud Monitoring"
          docsLink="https://grafana.com/docs/grafana/latest/datasources/google-cloud-monitoring/"
          hasRequiredFields
        />
        <Divider />
        <ConnectionConfig {...this.props} onOptionsChange={this.handleOnOptionsChange}></ConnectionConfig>
        {config.secureSocksDSProxyEnabled && (
          <>
            <Divider />
            <ConfigSection
              title="Additional settings"
              description="Additional settings are optional settings that can be configured for more control over your data source. This includes Secure Socks Proxy."
              isCollapsible={true}
              isInitiallyOpen={options.jsonData.enableSecureSocksProxy !== undefined}
            >
              <SecureSocksProxySettings options={options} onOptionsChange={onOptionsChange} />
            </ConfigSection>
          </>
        )}
        <Divider />
        <div className="gf-form-group">
          <h5 className="section-heading">Advanced settings</h5>
          <div className="gf-form">
            <label className="gf-form-label width-12">Universe Domain</label>
            <input
              className="gf-form-input width-30"
              value={options.jsonData.universeDomain}
              onChange={(event) =>
                this.handleOnOptionsChange({
                  ...options,
                  jsonData: { ...options.jsonData, universeDomain: event.target.value },
                })
              }
              placeholder="googleapis.com"
            />
          </div>
        </div>
      </>
    );
  }
}
