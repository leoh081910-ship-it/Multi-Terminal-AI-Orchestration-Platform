import { useState } from 'react';
import type { TreeNode } from '../api/goalApi';

interface TaskTreeViewProps {
  tree: TreeNode;
  onNodeClick?: (node: TreeNode) => void;
}

const STATE_COLORS: Record<string, string> = {
  queued: '#ffee00',
  decomposing: '#a78bfa',
  awaiting_approval: '#ff9f43',
  running: '#00f2ea',
  done: '#00ffaa',
  verified: '#00ffaa',
  failed: '#ff0050',
  blocked: '#ff0050',
};

function TreeNodeItem({ node, depth, onNodeClick }: { node: TreeNode; depth: number; onNodeClick?: (n: TreeNode) => void }) {
  const [expanded, setExpanded] = useState(true);
  const hasChildren = node.children && node.children.length > 0;
  const color = STATE_COLORS[node.state] || '#888';

  return (
    <div>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          padding: '6px 0',
          paddingLeft: depth * 24,
          cursor: hasChildren ? 'pointer' : 'default',
        }}
        onClick={() => onNodeClick?.(node)}
      >
        {hasChildren ? (
          <span
            onClick={(e) => { e.stopPropagation(); setExpanded(!expanded); }}
            style={{ width: 20, textAlign: 'center', color: '#888', cursor: 'pointer', fontSize: 14, marginRight: 4 }}
          >
            {expanded ? '▼' : '▶'}
          </span>
        ) : (
          <span style={{ width: 20, textAlign: 'center', color: '#555', fontSize: 10, marginRight: 4 }}>●</span>
        )}
        <span
          style={{
            display: 'inline-block',
            width: 8,
            height: 8,
            borderRadius: '50%',
            background: color,
            marginRight: 8,
            boxShadow: `0 0 6px ${color}66`,
          }}
        />
        <span style={{ fontSize: 13, color: '#ddd', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {node.title || node.id}
        </span>
        <span style={{ fontSize: 11, color, marginLeft: 8, fontWeight: 600 }}>
          {node.state}
        </span>
        {node.assigned_agent_id && (
          <span style={{ fontSize: 10, color: '#888', marginLeft: 8 }}>
            @{node.assigned_agent_id}
          </span>
        )}
      </div>
      {expanded && hasChildren && node.children!.map((child) => (
        <TreeNodeItem key={child.id} node={child} depth={depth + 1} onNodeClick={onNodeClick} />
      ))}
    </div>
  );
}

export default function TaskTreeView({ tree, onNodeClick }: TaskTreeViewProps) {
  return (
    <div style={{
      background: 'rgba(20,22,30,0.6)',
      borderRadius: 12,
      border: '1px solid rgba(255,255,255,0.06)',
      padding: 16,
      maxHeight: 600,
      overflowY: 'auto',
    }}>
      <TreeNodeItem node={tree} depth={0} onNodeClick={onNodeClick} />
    </div>
  );
}
