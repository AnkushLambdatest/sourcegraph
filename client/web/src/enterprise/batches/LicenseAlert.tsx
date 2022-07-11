import React from 'react'

import { Alert, AlertProps } from '@sourcegraph/wildcard'

import { useQuery } from '@sourcegraph/http-client'

import { GetLicenseAndUsageInfoResult, GetLicenseAndUsageInfoVariables } from '../../graphql-operations'

import { GET_LICENSE_AND_USAGE_INFO } from './list/backend'

export interface LicenseAlertProps {
    variant?: AlertProps['variant']
    // By default, the license is enough to determine to display alert. There may be cases where additional information
    // is needed to determine to show the alert.
    additionalCondition?: boolean
    // Allows the ability to apply additional logic to the parent component (such as disabling a button).
    onLicenseRetrieved?: (data: GetLicenseAndUsageInfoResult) => void
}

export const LicenseAlert: React.FunctionComponent<React.PropsWithChildren<LicenseAlertProps>> = ({
    variant = 'info',
    additionalCondition = true,
    onLicenseRetrieved,
}) => {
    const { data: licenseAndUsageInfo } = useQuery<GetLicenseAndUsageInfoResult, GetLicenseAndUsageInfoVariables>(
        GET_LICENSE_AND_USAGE_INFO,
        { onCompleted: onLicenseRetrieved }
    )

    if (!licenseAndUsageInfo) {
        return <></>
    }
    if (!licenseAndUsageInfo.batchChanges && !licenseAndUsageInfo.campaigns && additionalCondition) {
        return (
            <Alert variant={variant}>
                <div className="mb-2">
                    <strong>Your license only allows for 5 changesets per batch change</strong>
                </div>
                You are running a free version of batch changes. It is fully functional, however it will only generate 5
                changesets per batch change. If you would like to learn more about our pricing, contact us.
            </Alert>
        )
    }
    return <></>
}
