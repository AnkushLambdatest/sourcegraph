import React from 'react'

import { H3, H5, Input, Tooltip } from '@sourcegraph/wildcard'

export interface BatchChangeChangesetsHeaderProps {
    allSelected?: boolean
    toggleSelectAll?: () => void
    disabled?: boolean
}

export const BatchChangeChangesetsHeader: React.FunctionComponent<
    React.PropsWithChildren<BatchChangeChangesetsHeaderProps>
> = ({ allSelected, toggleSelectAll, disabled }) => (
    <>
        <span className="d-none d-md-block" />
        {toggleSelectAll && (
            <Tooltip
                content={
                    disabled ? 'You do not have permission to perform this operation' : 'Click to select all changesets'
                }
            >
                <Input
                    aria-label="Select all changesets"
                    type="checkbox"
                    checked={allSelected}
                    onChange={toggleSelectAll}
                    disabled={!!disabled}
                />
            </Tooltip>
        )}
        <H5 as={H3} className="p-2 pl-3 d-none d-md-block text-uppercase text-center text-nowrap">
            Status
        </H5>
        <H5 as={H3} className="p-2 d-none d-md-block text-uppercase text-nowrap">
            Changeset information
        </H5>
        <H5 as={H3} className="p-2 d-none d-md-block text-uppercase text-center text-nowrap">
            Check state
        </H5>
        <H5 as={H3} className="p-2 d-none d-md-block text-uppercase text-center text-nowrap">
            Review state
        </H5>
        <H5 as={H3} className="p-2 d-none d-md-block text-uppercase text-center text-nowrap">
            Changes
        </H5>
    </>
)
