document.addEventListener('DOMContentLoaded', () => {
    const apiKeyInput = document.getElementById('apiKey');
    const modelSelect = document.getElementById('modelSelect');
    const systemInstructionTextarea = document.getElementById('systemInstruction');
    const nextParagraphPromptTextarea = document.getElementById('nextParagraphPrompt');
    const storyOutputTextarea = document.getElementById('storyOutput');
    const generateBtn = document.getElementById('generateBtn');
    const revertLastParagraphBtn = document.getElementById('revertLastParagraphBtn'); 
    const clearAllBtn = document.getElementById('clearAllBtn');
    const loadingIndicator = document.getElementById('loadingIndicator');
    const errorDisplay = document.getElementById('errorDisplay');

    // New: Token display elements
    const currentRequestInputTokensDisplay = document.getElementById('currentRequestInputTokens');
    const currentRequestOutputTokensDisplay = document.getElementById('currentRequestOutputTokens');
    const accumulatedInputTokensDisplay = document.getElementById('accumulatedInputTokens'); 
    const accumulatedOutputTokensDisplay = document.getElementById('accumulatedOutputTokens'); 
    const accumulatedTokensDisplay = document.getElementById('accumulatedTokens'); 

    // No longer storing previous states for reverting, the button will directly modify the current text.

    // Load accumulated tokens from localStorage, default to 0 if not found
    let totalAccumulatedInputTokens = parseInt(localStorage.getItem('geminiTotalAccumulatedInputTokens') || '0', 10);
    let totalAccumulatedOutputTokens = parseInt(localStorage.getItem('geminiTotalAccumulatedOutputTokens') || '0', 10);
    let totalAccumulatedTokens = totalAccumulatedInputTokens + totalAccumulatedOutputTokens;

    const GEMINI_API_BASE_URL = 'https://generativelanguage.googleapis.com/v1beta/models/';

    const defaultSystemInstruction = `You are a skilled story writer.
Continue the story one paragraph at a time, keeping the tone consistent.
Ensure the new paragraph naturally follows the existing text and incorporates the given prompt for the next part of the story.
Use the same language as input or previous paragraph.`;
    // Load saved settings from localStorage
    apiKeyInput.value = localStorage.getItem('geminiApiKey') || '';
    modelSelect.value = localStorage.getItem('geminiModel') || 'gemini-2.5-flash-lite';
    systemInstructionTextarea.value = localStorage.getItem('geminiSystemInstruction') || defaultSystemInstruction;
    nextParagraphPromptTextarea.value = localStorage.getItem('geminiNextParagraphPrompt') || '';
    storyOutputTextarea.value = localStorage.getItem('geminiStoryOutput') || ''; 
    
    // Display loaded accumulated tokens
    accumulatedInputTokensDisplay.textContent = totalAccumulatedInputTokens;
    accumulatedOutputTokensDisplay.textContent = totalAccumulatedOutputTokens;
    if (accumulatedTokensDisplay) { 
        accumulatedTokensDisplay.textContent = totalAccumulatedTokens;
    }

    // Initialize button state based on loaded story content.
    revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim();

    // Save settings to localStorage on change
    apiKeyInput.addEventListener('input', () => localStorage.setItem('geminiApiKey', apiKeyInput.value));
    modelSelect.addEventListener('change', () => localStorage.setItem('geminiModel', modelSelect.value));
    systemInstructionTextarea.addEventListener('input', () => localStorage.setItem('geminiSystemInstruction', systemInstructionTextarea.value));
    nextParagraphPromptTextarea.addEventListener('input', () => localStorage.setItem('geminiNextParagraphPrompt', nextParagraphPromptTextarea.value));
    
    // Save story on manual input and update button state
    storyOutputTextarea.addEventListener('input', () => {
        localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value);
        revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim(); // Update button state on manual edit
    });


    generateBtn.addEventListener('click', generateParagraph);
    revertLastParagraphBtn.addEventListener('click', removeLastParagraph); // This function will now remove the last paragraph
    clearAllBtn.addEventListener('click', clearAllContents);

    function clearAllContents() {
        if (!confirm('Are you sure you want to clear all contents and settings (except API key)? This cannot be undone.')) {
            return;
        }

        modelSelect.value = 'gemini-2.5-flash-lite'; 
        systemInstructionTextarea.value = defaultSystemInstruction; 
        nextParagraphPromptTextarea.value = '';
        storyOutputTextarea.value = '';

        revertLastParagraphBtn.disabled = true;

        // Clear localStorage (except API key)
        localStorage.removeItem('geminiModel');
        localStorage.removeItem('geminiSystemInstruction');
        localStorage.removeItem('geminiNextParagraphPrompt');
        localStorage.removeItem('geminiStoryOutput'); 
        localStorage.removeItem('geminiTotalAccumulatedInputTokens'); 
        localStorage.removeItem('geminiTotalAccumulatedOutputTokens'); 

        // New: Clear token displays
        totalAccumulatedInputTokens = 0; 
        totalAccumulatedOutputTokens = 0; 
        totalAccumulatedTokens = 0; 
        currentRequestInputTokensDisplay.textContent = '0';
        currentRequestOutputTokensDisplay.textContent = '0';
        if (accumulatedInputTokensDisplay) { accumulatedInputTokensDisplay.textContent = '0'; }
        if (accumulatedOutputTokensDisplay) { accumulatedInputTokensDisplay.textContent = '0'; }
        if (accumulatedTokensDisplay) { 
            accumulatedTokensDisplay.textContent = '0';
        }

        showError(''); 
    }

    // This function is now responsible for removing the last paragraph directly from the textbox.
    function removeLastParagraph() {
        let currentStory = storyOutputTextarea.value.trim();
        if (!currentStory) {
            revertLastParagraphBtn.disabled = true;
            return;
        }

        // Split by two or more newlines to identify distinct paragraphs.
        // Trim each part and filter out any empty strings resulting from the split.
        let paragraphs = currentStory.split(/\n\n/).map(p => p.trim()).filter(p => p !== '');

        if (paragraphs.length > 0) {
            paragraphs.pop(); // Remove the last actual paragraph
            storyOutputTextarea.value = paragraphs.join('\n\n');
            localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value);
            
            // Re-evaluate button state based on the new content
            revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim();
            storyOutputTextarea.scrollTop = storyOutputTextarea.scrollHeight;
        } else {
            // If there were no discernible paragraphs left after splitting/filtering
            storyOutputTextarea.value = '';
            localStorage.setItem('geminiStoryOutput', '');
            revertLastParagraphBtn.disabled = true;
        }
    }

    async function generateParagraph() {
        const apiKey = apiKeyInput.value.trim();
        const selectedModel = modelSelect.value;
        const systemInstruction = systemInstructionTextarea.value.trim();
        const currentStory = storyOutputTextarea.value.trim();
        const nextParagraphPrompt = nextParagraphPromptTextarea.value.trim();

        if (!apiKey) {
            showError('Please enter your Gemini API Key.');
            return;
        }

        generateBtn.disabled = true;
        revertLastParagraphBtn.disabled = true; // Disable during generation
        loadingIndicator.classList.remove('hidden');
        showError(''); 

        currentRequestInputTokensDisplay.textContent = 'Calculating...'; 
        currentRequestOutputTokensDisplay.textContent = 'Calculating...'; 
        
        let userPrompt = '';
        if (currentStory === '') {
            userPrompt = `Start a new story. The first paragraph should be about: ${nextParagraphPrompt}`;
        } else {
            userPrompt = `Here is the story so far:\n\n${currentStory}\n\nWhat should happen next is: ${nextParagraphPrompt}\n\nContinue the story with ONE new paragraph, making sure it logically follows the previous text and incorporates the "what should happen next" prompt.`;
        }
        
        const requestBody = {
            contents: [{
                role: 'user',
                parts: [{ text: userPrompt }]
            }],
            generationConfig: {
                temperature: 0.9,
                topP: 1,
                topK: 1,
                maxOutputTokens: 500, 
            },
        };

        if (systemInstruction) {
            requestBody.systemInstruction = {
                parts: [{ text: systemInstruction }]
            };
        }

        try {
            const response = await fetch(`${GEMINI_API_BASE_URL}${selectedModel}:generateContent?key=${apiKey}`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody),
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error ? errorData.error.message : response.statusText);
            }

            const data = await response.json();
            const generatedText = data.candidates[0]?.content?.parts[0]?.text;

            const promptTokens = data.usageMetadata?.promptTokenCount || 0;
            const candidateTokens = data.usageMetadata?.candidatesTokenCount || 0; 

            currentRequestInputTokensDisplay.textContent = promptTokens;
            currentRequestOutputTokensDisplay.textContent = candidateTokens;
            
            totalAccumulatedInputTokens += promptTokens;
            totalAccumulatedOutputTokens += candidateTokens;
            totalAccumulatedTokens = totalAccumulatedInputTokens + totalAccumulatedOutputTokens; 

            accumulatedInputTokensDisplay.textContent = totalAccumulatedInputTokens;
            accumulatedOutputTokensDisplay.textContent = totalAccumulatedOutputTokens;
            if (accumulatedTokensDisplay) { 
                accumulatedTokensDisplay.textContent = totalAccumulatedTokens;
            }

            localStorage.setItem('geminiTotalAccumulatedInputTokens', totalAccumulatedInputTokens.toString());
            localStorage.setItem('geminiTotalAccumulatedOutputTokens', totalAccumulatedOutputTokens.toString());

            if (generatedText) {
                if (storyOutputTextarea.value.trim() === '') {
                    storyOutputTextarea.value = generatedText.trim();
                } else {
                    storyOutputTextarea.value += '\n\n' + generatedText.trim();
                }
                localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value); 
                revertLastParagraphBtn.disabled = false; // Enable button as there's now content
                nextParagraphPromptTextarea.value = '';
                storyOutputTextarea.scrollTop = storyOutputTextarea.scrollHeight;
            } else {
                showError('No content generated. The model might have been blocked due to safety concerns or returned an empty response.');
                currentRequestInputTokensDisplay.textContent = '0'; 
                currentRequestOutputTokensDisplay.textContent = '0'; 
                // On empty generation, the story output remains unchanged from its state before this attempt.
                revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim(); // Re-evaluate based on current content
            }

        } catch (error) {
            console.error('Error calling Gemini API:', error);
            showError(`Failed to generate paragraph: ${error.message}`);
            currentRequestInputTokensDisplay.textContent = '0'; 
            currentRequestOutputTokensDisplay.textContent = '0'; 
            // On error, the story output remains unchanged from its state before this attempt.
            revertLastParagraphBtn.disabled = !storyOutputTextarea.value.trim(); // Re-evaluate based on current content
        } finally {
            generateBtn.disabled = false;
            loadingIndicator.classList.add('hidden');
        }
    }

    function showError(message) {
        if (message) {
            errorDisplay.textContent = message;
            errorDisplay.classList.remove('hidden');
        } else {
            errorDisplay.textContent = '';
            errorDisplay.classList.add('hidden');
        }
    }
});