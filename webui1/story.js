document.addEventListener('DOMContentLoaded', () => {
    const apiKeyInput = document.getElementById('apiKey');
    const modelSelect = document.getElementById('modelSelect');
    const systemInstructionTextarea = document.getElementById('systemInstruction');
    const nextParagraphPromptTextarea = document.getElementById('nextParagraphPrompt');
    const storyOutputTextarea = document.getElementById('storyOutput');
    const generateBtn = document.getElementById('generateBtn');
    const clearAllBtn = document.getElementById('clearAllBtn');
    const loadingIndicator = document.getElementById('loadingIndicator');
    const errorDisplay = document.getElementById('errorDisplay');

    // New: Token display elements
    const currentRequestInputTokensDisplay = document.getElementById('currentRequestInputTokens');
    const currentRequestOutputTokensDisplay = document.getElementById('currentRequestOutputTokens');
    const accumulatedInputTokensDisplay = document.getElementById('accumulatedInputTokens'); // New
    const accumulatedOutputTokensDisplay = document.getElementById('accumulatedOutputTokens'); // New
    const accumulatedTokensDisplay = document.getElementById('accumulatedTokens'); // Corrected: Get the new element

    // Load accumulated tokens from localStorage, default to 0 if not found
    let totalAccumulatedInputTokens = parseInt(localStorage.getItem('geminiTotalAccumulatedInputTokens') || '0', 10);
    let totalAccumulatedOutputTokens = parseInt(localStorage.getItem('geminiTotalAccumulatedOutputTokens') || '0', 10);
    let totalAccumulatedTokens = totalAccumulatedInputTokens + totalAccumulatedOutputTokens; // Calculate from the two new variables

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
    storyOutputTextarea.value = localStorage.getItem('geminiStoryOutput') || ''; // Load story from localStorage
    
    // Display loaded accumulated tokens
    accumulatedInputTokensDisplay.textContent = totalAccumulatedInputTokens;
    accumulatedOutputTokensDisplay.textContent = totalAccumulatedOutputTokens;
    if (accumulatedTokensDisplay) { // Check if the element exists before trying to set textContent
        accumulatedTokensDisplay.textContent = totalAccumulatedTokens;
    }

    // Save settings to localStorage on change
    apiKeyInput.addEventListener('input', () => localStorage.setItem('geminiApiKey', apiKeyInput.value));
    modelSelect.addEventListener('change', () => localStorage.setItem('geminiModel', modelSelect.value));
    systemInstructionTextarea.addEventListener('input', () => localStorage.setItem('geminiSystemInstruction', systemInstructionTextarea.value));
    nextParagraphPromptTextarea.addEventListener('input', () => localStorage.setItem('geminiNextParagraphPrompt', nextParagraphPromptTextarea.value));
    storyOutputTextarea.addEventListener('input', () => localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value)); // Save story on manual input


    generateBtn.addEventListener('click', generateParagraph);
    clearAllBtn.addEventListener('click', clearAllContents);

    function clearAllContents() {
        if (!confirm('Are you sure you want to clear all contents and settings (except API key)? This cannot be undone.')) {
            return;
        }

        // Clear input fields (except API key)
        // apiKeyInput.value = ''; // Do NOT clear API key
        modelSelect.value = 'gemini-2.5-flash-lite'; // Reset to default model
        systemInstructionTextarea.value = defaultSystemInstruction; // Reset to default instruction
        nextParagraphPromptTextarea.value = '';
        storyOutputTextarea.value = '';

        // Clear localStorage (except API key)
        // localStorage.removeItem('geminiApiKey'); // Do NOT remove API key from localStorage
        localStorage.removeItem('geminiModel');
        localStorage.removeItem('geminiSystemInstruction');
        localStorage.removeItem('geminiNextParagraphPrompt');
        localStorage.removeItem('geminiStoryOutput'); // Clear story from storage
        localStorage.removeItem('geminiTotalAccumulatedInputTokens'); // Clear accumulated input tokens
        localStorage.removeItem('geminiTotalAccumulatedOutputTokens'); // Clear accumulated output tokens
        // localStorage.removeItem('geminiTotalAccumulatedTokens'); // This is derived, no need to remove separately

        // New: Clear token displays
        totalAccumulatedInputTokens = 0; // Reset variable
        totalAccumulatedOutputTokens = 0; // Reset variable
        totalAccumulatedTokens = 0; // Reset variable
        currentRequestInputTokensDisplay.textContent = '0';
        currentRequestOutputTokensDisplay.textContent = '0';
        accumulatedInputTokensDisplay.textContent = '0';
        accumulatedOutputTokensDisplay.textContent = '0';
        if (accumulatedTokensDisplay) { // Check if the element exists
            accumulatedTokensDisplay.textContent = '0';
        }

        showError(''); // Clear any displayed errors
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
        loadingIndicator.classList.remove('hidden');
        showError(''); // Clear previous errors

        currentRequestInputTokensDisplay.textContent = 'Calculating...'; // New: Indicate token calculation
        currentRequestOutputTokensDisplay.textContent = 'Calculating...'; // New: Indicate token calculation

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
                maxOutputTokens: 500, // Adjust as needed for paragraph length
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

            // New: Extract and display token usage
            const promptTokens = data.usageMetadata?.promptTokenCount || 0;
            const candidateTokens = data.usageMetadata?.candidatesTokenCount || 0; 

            currentRequestInputTokensDisplay.textContent = promptTokens;
            currentRequestOutputTokensDisplay.textContent = candidateTokens;
            
            // Accumulate tokens separately
            totalAccumulatedInputTokens += promptTokens;
            totalAccumulatedOutputTokens += candidateTokens;
            totalAccumulatedTokens = totalAccumulatedInputTokens + totalAccumulatedOutputTokens; // Recalculate total

            // Update display of accumulated tokens
            accumulatedInputTokensDisplay.textContent = totalAccumulatedInputTokens;
            accumulatedOutputTokensDisplay.textContent = totalAccumulatedOutputTokens;
            if (accumulatedTokensDisplay) { // Check if the element exists
                accumulatedTokensDisplay.textContent = totalAccumulatedTokens;
            }

            // Save updated accumulated tokens
            localStorage.setItem('geminiTotalAccumulatedInputTokens', totalAccumulatedInputTokens.toString());
            localStorage.setItem('geminiTotalAccumulatedOutputTokens', totalAccumulatedOutputTokens.toString());
            // localStorage.setItem('geminiTotalAccumulatedTokens', totalAccumulatedTokens.toString()); // Derived, no need to store

            if (generatedText) {
                if (storyOutputTextarea.value.trim() === '') {
                    storyOutputTextarea.value = generatedText.trim();
                } else {
                    storyOutputTextarea.value += '\n\n' + generatedText.trim();
                }
                localStorage.setItem('geminiStoryOutput', storyOutputTextarea.value); // Save updated story to localStorage
                // Clear the next paragraph prompt after generation
                nextParagraphPromptTextarea.value = '';
                // Scroll to the bottom of the story output
                storyOutputTextarea.scrollTop = storyOutputTextarea.scrollHeight;
            } else {
                showError('No content generated. The model might have been blocked due to safety concerns or returned an empty response.');
                currentRequestInputTokensDisplay.textContent = '0'; // New: Reset token display on empty generation
                currentRequestOutputTokensDisplay.textContent = '0'; // New: Reset token display on empty generation
            }

        } catch (error) {
            console.error('Error calling Gemini API:', error);
            showError(`Failed to generate paragraph: ${error.message}`);
            currentRequestInputTokensDisplay.textContent = '0'; // New: Reset token display on error
            currentRequestOutputTokensDisplay.textContent = '0'; // New: Reset token display on error
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