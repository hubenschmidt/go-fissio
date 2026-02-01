import { Show, onMount } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { chat } from '../lib/stores/chat';
import PipelineEditor from '../lib/components/PipelineEditor';
import type { PipelineInfo } from '../lib/types';

export default function Composer() {
  const navigate = useNavigate();

  onMount(() => {
    chat.connect();
  });

  const handlePipelineUpdate = (config: PipelineInfo) => {
    chat.setPipelineConfig(config);
  };

  const handleSavePipeline = async (config: PipelineInfo) => {
    await chat.savePipeline(config);
    chat.setSelectedPipeline(config.id);
    navigate('/');
  };

  const createNewAgent = () => {
    const id = `custom_${Date.now()}`;
    const blank: PipelineInfo = {
      id,
      name: 'New Agent',
      description: '',
      nodes: [
        { id: 'llm1', node_type: 'llm', model: null, prompt: 'You are a helpful assistant.' }
      ],
      edges: [
        { from: 'input', to: 'llm1' },
        { from: 'llm1', to: 'output' }
      ]
    };
    chat.setPipelineConfig(blank);
  };

  return (
    <Show
      when={chat.pipelineConfig()}
      fallback={
        <div class="no-config">
          <p>No agent config loaded</p>
          <div class="no-config-actions">
            <button class="primary" onClick={createNewAgent}>Create New Agent</button>
            <button onClick={() => navigate('/')}>Back to Chat</button>
          </div>
        </div>
      }
    >
      {(config) => (
        <PipelineEditor
          config={config()}
          models={chat.models()}
          templates={chat.templates()}
          availableTools={chat.availableTools()}
          onUpdate={handlePipelineUpdate}
          onSave={handleSavePipeline}
        />
      )}
    </Show>
  );
}
