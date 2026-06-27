import {
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Chip,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { ComboRegion, ComboSlice } from '../types';

// RegionPanel 组件属性
interface RegionPanelProps {
  regions: ComboRegion[];
  onSliceClick: (region: ComboRegion, slice: ComboSlice) => void;
}

/** 区域折叠面板 — 以手风琴形式展示区域及其包含的片段 */
export function RegionPanel({ regions, onSliceClick }: RegionPanelProps) {
  return (
    <>
      {regions.map((r) => (
        <Accordion key={r.id} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="body2">
              {r.name} ({r.slices.length})
            </Typography>
          </AccordionSummary>
          <AccordionDetails
            sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}
          >
            {r.slices.map((s) => (
              <Chip
                key={s.id}
                label={s.content}
                size="small"
                variant="outlined"
                onClick={() => onSliceClick(r, s)}
                sx={{ cursor: 'pointer' }}
              />
            ))}
          </AccordionDetails>
        </Accordion>
      ))}
    </>
  );
}
